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

type ListProjectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListProjectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProjectsLogic {
	return &ListProjectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListProjectsLogic) ListProjects() (resp *types.ProjectListResponse, err error) {
	projects := l.svcCtx.ProjectManager.ListProjects()

	result := make([]types.ProjectResponse, len(projects))
	for i, p := range projects {
		result[i] = types.ProjectResponse{
			Path:         p.Path,
			Sessions:     p.Sessions,
			LastModified: p.LastModified.Format(time.RFC3339),
		}
	}

	return &types.ProjectListResponse{Projects: result}, nil
}
