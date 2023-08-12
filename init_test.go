package tmpdocker

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

var tmpd *TmpDocker

func TestMain(m *testing.M) {
	tmpd = &TmpDocker{
		ServiceName: "tmpdocker_test",
		KeepAlive:   caddy.Duration(time.Minute),
	}

	const image = "nginx:1.19.6-alpine@sha256:c2ce58e024275728b00a554ac25628af25c54782865b3487b11c21cafb7fabda"
	if err := runCmd("docker", "pull", image); err != nil {
		panic(err)
	}
	if err := runCmd("docker",
		"service", "create",
		"--name", tmpd.ServiceName,
		"--replicas", "0",
		"-p", "8081:80",
		image,
	); err != nil {
		panic(err)
	}
	defer func() {
		if err := runCmd("docker", "service", "rm", tmpd.ServiceName); err != nil {
			panic(err)
		}
	}()

	ctx := caddy.Context{Context: context.Background()}
	if err := tmpd.Provision(ctx); err != nil {
		panic(err)
	}
	if err := tmpd.Validate(); err != nil {
		panic(err)
	}
	m.Run()
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
