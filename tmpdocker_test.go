package tmpdocker

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"golang.org/x/sync/errgroup"
)

func TestCaddy(t *testing.T) {
	defer err2.Catch(func(err error) {
		t.Error(err)
	})

	try.To(runCmd("go", "build", "-o", "caddy", "./cmd/caddy"))
	cmd := exec.Command(
		"./caddy",
		"run",
		"--config", "./cmd/caddy/Caddyfile",
		"--adapter", "caddyfile",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	try.To(cmd.Start())
	defer cmd.Process.Kill()

	time.Sleep(2 * time.Second)

	eg := new(errgroup.Group)

	for i := 0; i < 2; i++ {
		eg.Go(httpGet)
	}

	try.To(eg.Wait())

	ctx := context.Background()
	ds := try.To1(tmpd.GetTmpService(ctx))

	{
		count := try.To1(tmpd.GetRunning(ctx, ds.ID))
		assert.Equal(count, 1)
	}

	{
		<-time.After(time.Minute * 12 / 10)
		count := try.To1(tmpd.GetRunning(ctx, ds.ID))
		assert.Equal(count, 0)
	}

	{
		try.To(httpGet())
		count := try.To1(tmpd.GetRunning(ctx, ds.ID))
		assert.Equal(count, 1)
	}

}

func httpGet() (err error) {
	defer err2.Handle(&err)
	resp := try.To1(http.Get("http://127.0.0.1:8080"))
	if resp.StatusCode != http.StatusOK {
		err2.Throwf("status code is %d", resp.StatusCode)
	}
	return
}
