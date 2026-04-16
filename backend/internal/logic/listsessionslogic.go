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

type ListSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionsLogic {
	return &ListSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSessionsLogic) ListSessions(req *types.SessionListRequest) (resp *types.SessionListResponse, err error) {
	sessions := l.svcCtx.SessionManager.ListSessions(req.ProjectPath)
	var list []types.SessionResponse
	for _, s := range sessions {
		list = append(list, types.SessionResponse{
			ID:          s.ID,
			ProjectPath: s.ProjectPath,
			Model:       s.Model,
			State:       string(s.State),
			CreatedAt:   s.CreatedAt.Format(time.RFC3339),
			BlockedOn:   s.BlockedOn,
			BlockedTool: s.BlockedTool,
		})
	}
	return &types.SessionListResponse{Sessions: list}, nil
}
