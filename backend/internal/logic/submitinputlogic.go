// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"agent-base/pkg/security"
	"context"
	"errors"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitInputLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitInputLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitInputLogic {
	return &SubmitInputLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitInputLogic) SubmitInput(req *types.SubmitInputRequest) (*types.SessionResponse, error) {
	mode := req.Mode
	if mode == "" {
		mode = security.PlanMode
	}

	if err := l.svcCtx.SessionManager.SubmitInput(req.ID, req.Input, mode); err != nil {
		return nil, err
	}

	sess := l.svcCtx.SessionManager.GetSession(req.ID)
	if sess == nil {
		return nil, errors.New("session not found")
	}

	go RunAgent(l.svcCtx.EngineManager, l.svcCtx.SessionManager, l.svcCtx.EventBus, sess)

	return &types.SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, nil
}
