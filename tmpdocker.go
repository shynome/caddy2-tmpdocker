// Copyright 2015 Matthew Holt and The Caddy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tmpdocker

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(TmpDocker{})
}

// TmpDocker is a middleware which can rewrite HTTP requests.
type TmpDocker struct {
	ServiceName  string         `json:"service_name,omitempty"`
	KeepAlive    caddy.Duration `json:"keep_alive,omitempty"`
	ScaleTimeout caddy.Duration `json:"scale_timeout,omitempty"`
	DockerHost   string         `json:"docker_host,omitempty"`

	checkDuration  time.Duration
	lastActiveTime *int64
	client         *client.Client
	lock           *sync.Cond

	timer     *time.Ticker
	timerStop chan bool

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (TmpDocker) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.tmpdocker",
		New: func() caddy.Module { return new(TmpDocker) },
	}
}

// Provision sets up tmpd.
func (tmpd *TmpDocker) Provision(ctx caddy.Context) error {
	if tmpd.KeepAlive == 0 {
		tmpd.KeepAlive = caddy.Duration(5 * time.Minute)
	}
	if tmpd.lastActiveTime == nil {
		zero := int64(0)
		tmpd.lastActiveTime = &zero
	}
	if tmpd.ScaleTimeout == 0 {
		tmpd.ScaleTimeout = caddy.Duration(10 * time.Second)
	}
	tmpd.checkDuration = time.Duration(tmpd.KeepAlive / 10)
	tmpd.logger = ctx.Logger(tmpd)
	return nil
}

// Validate validates tmpd.
func (tmpd *TmpDocker) Validate() (err error) {
	if tmpd.ServiceName == "" {
		return fmt.Errorf("docker service_name is required")
	}
	if time.Duration(tmpd.KeepAlive) < time.Minute {
		return fmt.Errorf("freeze_timeout must greater than 1m")
	}
	if tmpd.DockerHost == "" {
		if tmpd.client, err = client.NewEnvClient(); err != nil {
			return err
		}
	} else {
		if tmpd.client, err = client.NewClient(tmpd.DockerHost, "", nil, nil); err != nil {
			return err
		}
	}
	// if _, err := tmpd.GetTmpService(); err != nil {
	// 	return err
	// }
	return nil
}

func (tmpd *TmpDocker) newCheckTimer() {
	for {
		select {
		case <-tmpd.timerStop:
			tmpd.resetStatus()
			tmpd.timer.Stop()
			tmpd.timer = nil
			tmpd.timerStop = nil
			go tmpd.StopDockerService()
			return
		case <-tmpd.timer.C:
			duration := time.Now().UnixNano() - atomic.LoadInt64(tmpd.lastActiveTime)
			tmpd.logger.Debug("check duration",
				zap.String("docker service", tmpd.ServiceName),
				zap.Int64("duration", duration/int64(time.Second)),
				zap.Int64("freeze", int64(tmpd.KeepAlive)/int64(time.Second)),
			)
			if duration > int64(tmpd.KeepAlive) {
				tmpd.timer.Stop()
				go func() { tmpd.timerStop <- true }()
			}
		}
	}
}

func (tmpd *TmpDocker) resetStatus() {
	tmpd.lock = nil
	atomic.StoreInt64(tmpd.lastActiveTime, 0)
}

func (tmpd *TmpDocker) updateLastActiveUnixTime(t int64) {
	lastActiveTime := atomic.LoadInt64(tmpd.lastActiveTime)
	atomic.StoreInt64(tmpd.lastActiveTime, t)
	if lastActiveTime != 0 { // already have a timer
		return
	}
	tmpd.timer = time.NewTicker(tmpd.checkDuration)
	tmpd.timerStop = make(chan bool)
	go tmpd.newCheckTimer()
}
func (tmpd *TmpDocker) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	t := time.Now().UnixNano()
	lat := atomic.LoadInt64(tmpd.lastActiveTime)

	if tmpd.lock != nil {
		if lat == 0 {
			lock := tmpd.lock
			lock.L.Lock()
			defer lock.L.Unlock()
			for atomic.LoadInt64(tmpd.lastActiveTime) == 0 {
				lock.Wait()
			}
		}
		defer func() { go tmpd.updateLastActiveUnixTime(t) }()
		return next.ServeHTTP(w, r)
	}

	lock := sync.NewCond(&sync.Mutex{})
	tmpd.lock = lock
	defer lock.Broadcast()
	if err := tmpd.ScaleDockerService(); err != nil {
		// recovery
		tmpd.resetStatus()
		return err
	}
	tmpd.updateLastActiveUnixTime(t)
	return next.ServeHTTP(w, r)
}

// TmpService v
type TmpService struct {
	ID          string
	Replicas    uint64
	ServiceSpec swarm.ServiceSpec
	Version     swarm.Version
}

// GetTmpService v
func (tmpd TmpDocker) GetTmpService() (*TmpService, error) {
	client := tmpd.client
	f := filters.NewArgs()
	f.Add("name", tmpd.ServiceName)
	slst, err := client.ServiceList(context.Background(), types.ServiceListOptions{Filters: f})
	if err != nil {
		return nil, err
	}
	if len(slst) == 0 {
		return nil, fmt.Errorf("docker service %s has not preset", tmpd.ServiceName)
	}
	s := slst[0]
	if s.Spec.Mode.Replicated == nil {
		return nil, fmt.Errorf("scale can only be used with replicated mode")
	}
	replicas := s.Spec.Mode.Replicated.Replicas

	return &TmpService{
		ID:          s.ID,
		Replicas:    *replicas,
		Version:     s.Meta.Version,
		ServiceSpec: s.Spec,
	}, nil
}

// GetRunning node length
func (tmpd TmpDocker) GetRunning(serviceID string) (count int, err error) {
	client := tmpd.client
	f := filters.NewArgs()
	f.Add("service", serviceID)
	tasks, err := client.TaskList(context.Background(), types.TaskListOptions{Filters: f})
	if err != nil {
		return
	}
	for _, task := range tasks {
		if task.DesiredState == swarm.TaskStateRunning && task.Status.State == swarm.TaskStateRunning {
			count++
			break // don't need range all
		}
	}
	return
}

// ScaleDockerService use docker
func (tmpd TmpDocker) ScaleDockerService() error {
	client := tmpd.client
	ds, err := tmpd.GetTmpService()
	if err != nil {
		return err
	}
	count, err := tmpd.GetRunning(ds.ID)
	if err != nil {
		return err
	}
	if ds.Replicas != 0 && uint64(count) == ds.Replicas {
		return nil
	}
	if err != nil {
		return err
	}

	replicas := uint64(1)
	ds.ServiceSpec.Mode.Replicated.Replicas = &replicas

	_, err = client.ServiceUpdate(
		context.Background(),
		ds.ID,
		ds.Version,
		ds.ServiceSpec,
		types.ServiceUpdateOptions{},
	)
	tmpd.logger.Info("scale docker service",
		zap.String("name", tmpd.ServiceName),
	)
	if err != nil {
		return err
	}
	for timeoutPoint := time.Now().Add(time.Duration(tmpd.ScaleTimeout)); ; {
		count, err := tmpd.GetRunning(ds.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			break
		}
		if !time.Now().Before(timeoutPoint) {
			return fmt.Errorf("start docker service %v fail, because wake timeout", tmpd.ServiceName)
		}
		time.Sleep(time.Second)
	}
	return nil
}

// StopDockerService use docker
func (tmpd TmpDocker) StopDockerService() error {
	tmpd.logger.Info("stop docker service",
		zap.String("name", tmpd.ServiceName),
	)
	client := tmpd.client
	ds, err := tmpd.GetTmpService()
	if err != nil {
		return err
	}
	if ds.Replicas == 0 {
		return nil
	}
	replicas := uint64(0)
	ds.ServiceSpec.Mode.Replicated.Replicas = &replicas

	_, err = client.ServiceUpdate(
		context.Background(),
		ds.ID,
		ds.Version,
		ds.ServiceSpec,
		types.ServiceUpdateOptions{},
	)
	if err != nil {
		return err
	}
	return nil
}

// Interface guard
var _ caddyhttp.MiddlewareHandler = (*TmpDocker)(nil)
