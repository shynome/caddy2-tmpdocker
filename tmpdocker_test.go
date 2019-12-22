package tmpdocker

import (
	"github.com/caddyserver/caddy/v2"
	"testing"
	"time"
)

var tmpd = func() *TmpDocker {
	tmpd := &TmpDocker{
		ServiceName:    "ttt",
		FreezeDuration: caddy.Duration(5 * time.Minute),
	}
	if err := tmpd.Validate(); err != nil {
		panic(err)
	}
	return tmpd
}()

func TestScale(t *testing.T) {
	err := tmpd.ScaleDockerService()
	if err != nil {
		t.Error(err)
		return
	}
}

func TestStop(t *testing.T) {
	err := tmpd.StopDockerService()
	if err != nil {
		t.Error(err)
		return
	}
}
