package tmpdocker

import (
	"testing"
)

func TestScale(t *testing.T) {
	a := TmpDocker{ServiceName: "ttt"}
	err := a.ScaleDockerService()
	if err != nil {
		t.Error(err)
		return
	}
}

func TestStop(t *testing.T) {
	a := TmpDocker{ServiceName: "ttt"}
	err := a.StopDockerService()
	if err != nil {
		t.Error(err)
		return
	}
}
