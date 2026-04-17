// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/systems/session"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnblockSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnblockSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnblockSessionLogic {
	return &UnblockSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnblockSessionLogic) UnblockSession(req *types.UnblockRequest) (*types.SessionResponse, error) {
	sess := l.svcCtx.SessionManager.GetSession(req.ID)
	if sess == nil {
		return nil, errors.New("session not found")
	}

	if sess.State != session.StateBlocked {
		return nil, errors.New("session not in blocked state")
	}

	decision := session.PermissionDecision{
		Approved:   req.Approved,
		AddAllowed: req.AddAllowed,
	}

	if err := l.svcCtx.SessionManager.ClearBlockedState(req.ID); err != nil {
		return nil, err
	}

	if err := l.svcCtx.SessionManager.SubmitPermissionDecision(req.ID, decision); err != nil {
		return nil, err
	}

	sess = l.svcCtx.SessionManager.GetSession(req.ID)
	return &types.SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, nil
}
