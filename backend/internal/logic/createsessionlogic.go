// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSessionLogic {
	return &CreateSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSessionLogic) CreateSession(req *types.CreateSessionRequest) (*types.SessionResponse, error) {
	model := req.Model
	if model == "" {
		model = "glm-5"
	}

	sess := l.svcCtx.SessionManager.CreateSession(req.ProjectPath, model)

	l.svcCtx.ProjectManager.AddSession(req.ProjectPath, sess.ID)

	return &types.SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, nil
}
