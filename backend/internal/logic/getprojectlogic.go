// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProjectLogic {
	return &GetProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProjectLogic) GetProject(req *types.GetProjectRequest) (resp *types.ProjectResponse, err error) {
	p := l.svcCtx.ProjectManager.GetProject(req.Name)
	if p == nil {
		return nil, errors.New("project not found")
	}

	return &types.ProjectResponse{
		Path:     p.Path,
		Sessions: p.Sessions,
	}, nil
}
