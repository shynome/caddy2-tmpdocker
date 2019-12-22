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
	ServiceName    string         `json:"service_name,omitempty"`
	FreezeDuration caddy.Duration `json:"freeze_timeout,omitempty"`
	DockerHost     string         `json:"docker_host,omitempty"`

	checkDuration  time.Duration
	lastActiveTime int64
	client         *client.Client
	lock           *sync.Cond

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
	if tmpd.FreezeDuration == 0 {
		tmpd.FreezeDuration = caddy.Duration(20 * time.Minute)
	}
	tmpd.checkDuration = time.Duration(tmpd.FreezeDuration / 10)
	tmpd.logger = ctx.Logger(tmpd)
	return nil
}

// Validate validates tmpd.
func (tmpd *TmpDocker) Validate() (err error) {
	if tmpd.ServiceName == "" {
		return fmt.Errorf("docker service_name is required")
	}
	if time.Duration(tmpd.FreezeDuration) < 5*time.Minute {
		return fmt.Errorf("freeze_timeout must greater than 5m")
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
	if _, err := tmpd.GetTmpService(); err != nil {
		return err
	}
	return nil
}

func (tmpd TmpDocker) updateLastActiveUnixTime(t int64) {
	tmpd.lastActiveTime = t
	if tmpd.lastActiveTime != 0 { // already have a timer
		return
	}
	for {
		time.Sleep(tmpd.checkDuration)
		duration := time.Now().Unix() - tmpd.lastActiveTime
		if duration > int64(tmpd.FreezeDuration) {
			tmpd.lastActiveTime = 0
			tmpd.lock = nil
			go tmpd.StopDockerService()
			break
		}
	}
}
func (tmpd TmpDocker) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if tmpd.lastActiveTime == 0 {
		if tmpd.lock == nil {
			m := sync.Mutex{}
			tmpd.lock = sync.NewCond(&m)
			m.Lock()
			if err := tmpd.ScaleDockerService(); err != nil {
				return err
			}
			m.Unlock()
		} else {
			tmpd.lock.Wait()
		}
	}
	go tmpd.updateLastActiveUnixTime(time.Now().Unix())
	return next.ServeHTTP(w, r)
}

type TmpService struct {
	ID          string
	Replicas    uint64
	ServiceSpec swarm.ServiceSpec
	Version     swarm.Version
}

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

// ScaleDockerService use docker
func (tmpd TmpDocker) ScaleDockerService() error {
	client := tmpd.client
	ds, err := tmpd.GetTmpService()
	if err != nil {
	}
	if ds.Replicas > 0 {
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
	if err != nil {
		return err
	}
	for i := 0; true; i++ {
		ds, err := tmpd.GetTmpService()
		if err != nil {
			return err
		}
		if ds.Replicas > 0 {
			break
		}
		if i > 5 {
			return fmt.Errorf("start docker service %v fail", tmpd.ServiceName)
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (tmpd TmpDocker) StopDockerService() error {
	client := tmpd.client
	ds, err := tmpd.GetTmpService()
	if err != nil {
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
