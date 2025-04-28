package tmpdocker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

// TmpService v
type TmpService struct {
	ID          string
	Replicas    uint64
	ServiceSpec swarm.ServiceSpec
	Version     swarm.Version
}

// GetTmpService v
func (tmpd TmpDocker) GetTmpService(ctx context.Context) (*TmpService, error) {
	client := tmpd.client
	f := filters.NewArgs()
	f.Add("name", tmpd.ServiceName)
	slst, err := client.ServiceList(ctx, types.ServiceListOptions{Filters: f})
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
func (tmpd TmpDocker) GetRunning(ctx context.Context, serviceID string) (count int, err error) {
	client := tmpd.client
	f := filters.NewArgs()
	f.Add("service", serviceID)
	tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: f})
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
func (tmpd TmpDocker) ScaleDockerService(ctx context.Context) error {

	client := tmpd.client
	ds, err := tmpd.GetTmpService(ctx)
	if err != nil {
		return err
	}
	count, err := tmpd.GetRunning(ctx, ds.ID)
	if err != nil {
		return err
	}
	if ds.Replicas != 0 && uint64(count) == ds.Replicas {
		return nil
	}

	replicas := uint64(1)
	ds.ServiceSpec.Mode.Replicated.Replicas = &replicas

	_, err = client.ServiceUpdate(
		ctx,
		ds.ID,
		ds.Version,
		ds.ServiceSpec,
		types.ServiceUpdateOptions{},
	)
	if err != nil {
		return err
	}
	for {
		count, err := tmpd.GetRunning(ctx, ds.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			break
		}
	}
	return nil
}

// StopDockerService use docker
func (tmpd TmpDocker) StopDockerService(ctx context.Context) error {
	client := tmpd.client
	ds, err := tmpd.GetTmpService(ctx)
	if err != nil {
		return err
	}
	if ds.Replicas == 0 {
		return nil
	}
	replicas := uint64(0)
	ds.ServiceSpec.Mode.Replicated.Replicas = &replicas

	_, err = client.ServiceUpdate(
		ctx,
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
