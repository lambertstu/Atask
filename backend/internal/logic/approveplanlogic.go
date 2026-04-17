// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"agent-base/pkg/security"
	"context"
	"errors"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/systems/session"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApprovePlanLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApprovePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApprovePlanLogic {
	return &ApprovePlanLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApprovePlanLogic) ApprovePlan(req *types.ApprovePlanRequest) (*types.SessionResponse, error) {
	sess := l.svcCtx.SessionManager.GetSession(req.ID)
	if sess == nil {
		return nil, errors.New("session not found")
	}

	if sess.PermissionMgr != nil {
		sess.PermissionMgr.SetMode(security.BuildMode)
	}

	if err := l.svcCtx.SessionManager.Transition(req.ID, session.StateProcessing, security.BuildMode); err != nil {
		return nil, err
	}

	go RunAgent(l.svcCtx.EngineManager, l.svcCtx.SessionManager, l.svcCtx.EventBus, sess)

	sess = l.svcCtx.SessionManager.GetSession(req.ID)
	return &types.SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, nil
}
