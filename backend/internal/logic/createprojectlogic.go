// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateProjectLogic {
	return &CreateProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateProjectLogic) CreateProject(req *types.CreateProjectRequest) (resp *types.ProjectResponse, err error) {
	p := l.svcCtx.ProjectManager.GetOrCreate(req.Path)

	return &types.ProjectResponse{
		Path:     p.Path,
		Sessions: p.Sessions,
	}, nil
}
