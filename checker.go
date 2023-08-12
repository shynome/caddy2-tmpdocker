package tmpdocker

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

func (tmpd *TmpDocker) StartChecker(ctx context.Context) {
	t := time.NewTicker(time.Duration(tmpd.KeepAlive / 10))
	defer t.Stop()

	keepAlive := int64(time.Duration(tmpd.KeepAlive) / time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tmpd.check(ctx, keepAlive)
		}
	}
}

func (tmpd *TmpDocker) check(ctx context.Context, keepAlive int64) {
	lat := atomic.LoadInt64(&tmpd.lastActiveTime)
	if lat <= 0 {
		return
	}

	duration := time.Now().Unix() - lat
	tmpd.logger.Debug("check duration",
		zap.String("docker service", tmpd.ServiceName),
		zap.Int64("duration", duration),
		zap.Int64("keepalive", keepAlive),
	)

	if duration < keepAlive {
		return
	}
	atomic.StoreInt64(&tmpd.lastActiveTime, 0)

	tmpd.logger.Info("stop docker service",
		zap.String("name", tmpd.ServiceName),
	)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(tmpd.ScaleTimeout))
	defer cancel()
	if err := tmpd.StopDockerService(ctx); err != nil {
		tmpd.logger.Warn("stop docker service failed",
			zap.String("name", tmpd.ServiceName),
			zap.Error(err),
		)
	}
}
