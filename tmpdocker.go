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

	logger *zap.Logger

	lastActiveTime int64 // unix time
	client         *client.Client
	cond           *sync.Cond
}

var _ caddyhttp.MiddlewareHandler = (*TmpDocker)(nil)

// CaddyModule returns the Caddy module information.
func (TmpDocker) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.tmpdocker",
		New: func() caddy.Module { return new(TmpDocker) },
	}
}

// Provision sets up tmpd.
func (tmpd *TmpDocker) Provision(ctx caddy.Context) error {
	tmpd.cond = sync.NewCond(&sync.RWMutex{})
	if tmpd.KeepAlive == 0 {
		tmpd.KeepAlive = caddy.Duration(5 * time.Minute)
	}
	if tmpd.ScaleTimeout == 0 {
		tmpd.ScaleTimeout = caddy.Duration(10 * time.Second)
	}
	tmpd.logger = ctx.Logger(tmpd)

	go tmpd.StartChecker(ctx)

	return nil
}

// Validate validates tmpd.
func (tmpd *TmpDocker) Validate() (err error) {
	if tmpd.ServiceName == "" {
		return fmt.Errorf("docker service_name is required")
	}
	if time.Duration(tmpd.KeepAlive) < time.Minute {
		return fmt.Errorf("keep_alive must greater than 1m")
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

const (
	statusScaling int64 = -1
)

func (tmpd *TmpDocker) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) (err error) {
	now := time.Now().Unix()
	defer func() {
		if err != nil {
			atomic.StoreInt64(&tmpd.lastActiveTime, 0)
			return
		}
		atomic.StoreInt64(&tmpd.lastActiveTime, now)
	}()

	lat := atomic.LoadInt64(&tmpd.lastActiveTime)

	if lat > 0 {
		return next.ServeHTTP(w, r)
	}

	if lat == statusScaling {
		c := tmpd.cond
		c.L.Lock()
		defer c.L.Unlock()
		for atomic.LoadInt64(&tmpd.lastActiveTime) == statusScaling {
			c.Wait()
		}
		return next.ServeHTTP(w, r)
	}

	atomic.StoreInt64(&tmpd.lastActiveTime, statusScaling)

	c := tmpd.cond
	defer c.Broadcast()
	c.L.Lock()
	defer c.L.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(tmpd.ScaleTimeout))
	defer cancel()
	tmpd.logger.Info("scale docker service",
		zap.String("name", tmpd.ServiceName),
	)
	if err := tmpd.ScaleDockerService(ctx); err != nil {
		tmpd.logger.Info("scale docker service failed",
			zap.String("name", tmpd.ServiceName),
			zap.Error(err),
		)
		return err
	}

	atomic.StoreInt64(&tmpd.lastActiveTime, 1)

	return next.ServeHTTP(w, r)
}
