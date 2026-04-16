// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSessionLogic {
	return &GetSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSessionLogic) GetSession(req *types.GetSessionRequest) (*types.SessionResponse, error) {
	sess := l.svcCtx.SessionManager.GetSession(req.ID)
	if sess == nil {
		return nil, errors.New("session not found")
	}

	return &types.SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
		BlockedOn:   sess.BlockedOn,
		BlockedTool: sess.BlockedTool,
	}, nil
}
