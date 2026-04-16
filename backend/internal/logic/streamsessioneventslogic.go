// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/types"
	"agent-base/pkg/events"

	"github.com/zeromicro/go-zero/core/logx"
)

type StreamSessionEventsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStreamSessionEventsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StreamSessionEventsLogic {
	return &StreamSessionEventsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StreamSessionEventsLogic) StreamSessionEvents(req *types.StreamSessionEventsRequest, client chan<- *types.SessionEvent) error {
	eventCh, subscriberID := l.svcCtx.EventBus.Subscribe(req.ID)
	defer l.svcCtx.EventBus.Unsubscribe(req.ID, subscriberID)

	for {
		select {
		case event := <-eventCh:
			client <- convertEvent(event)
		case <-l.ctx.Done():
			return nil
		}
	}
}

func convertEvent(e events.Event) *types.SessionEvent {
	return &types.SessionEvent{
		Type:      string(e.Type),
		SessionID: e.SessionID,
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Data:      e.Data,
	}
}
