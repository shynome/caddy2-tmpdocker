package tmpdocker

import (
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
	"testing"
	"time"
)

var tmpd = func() *TmpDocker {
	tmpd := &TmpDocker{
		ServiceName:   "ttt",
		FreezeTimeout: caddy.Duration(5 * time.Minute),
	}
	zero := int64(0)
	tmpd.lastActiveTime = &zero
	tmpd.logger = zap.NewNop()
	if err := tmpd.Validate(); err != nil {
		panic(err)
	}
	return tmpd
}()

func TestGetRunning(t *testing.T) {
	ds, _ := tmpd.GetTmpService()
	count, err := tmpd.GetRunning(ds.ID)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(count)
}

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
