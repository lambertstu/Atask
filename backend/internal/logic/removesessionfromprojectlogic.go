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

type RemoveSessionFromProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveSessionFromProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveSessionFromProjectLogic {
	return &RemoveSessionFromProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveSessionFromProjectLogic) RemoveSessionFromProject(req *types.RemoveSessionFromProjectRequest) (resp *types.ProjectResponse, err error) {
	p := l.svcCtx.ProjectManager.GetProject(req.Name)
	if p == nil {
		return nil, errors.New("project not found")
	}

	if err := l.svcCtx.ProjectManager.RemoveSession(p.Path, req.SessionID); err != nil {
		return nil, err
	}

	updated := l.svcCtx.ProjectManager.GetProject(req.Name)
	return &types.ProjectResponse{
		Path:         updated.Path,
		Sessions:     updated.Sessions,
		LastModified: updated.LastModified.Format(time.RFC3339),
	}, nil
}
