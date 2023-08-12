package tmpdocker

import (
	"context"
	"testing"
	"time"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestGetRunning(t *testing.T) {
	ctx := context.Background()
	ds, _ := tmpd.GetTmpService(ctx)
	count, err := tmpd.GetRunning(ctx, ds.ID)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(count)
}

func TestScale(t *testing.T) {
	ctx := context.Background()
	try.To(tmpd.ScaleDockerService(ctx))
	ds := try.To1(tmpd.GetTmpService(ctx))
	count := try.To1(tmpd.GetRunning(ctx, ds.ID))
	assert.Equal(count, 1)
}

func TestStop(t *testing.T) {
	ctx := context.Background()
	ds := try.To1(tmpd.GetTmpService(ctx))

	try.To(tmpd.ScaleDockerService(ctx))
	try.To(tmpd.StopDockerService(ctx))

	time.Sleep(3 * time.Second)
	count := try.To1(tmpd.GetRunning(ctx, ds.ID))
	assert.Equal(count, 0)
}
