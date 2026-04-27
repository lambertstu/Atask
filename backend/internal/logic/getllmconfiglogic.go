// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLLMConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLLMConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLLMConfigLogic {
	return &GetLLMConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLLMConfigLogic) GetLLMConfig() (resp *types.LLMConfigResponse, err error) {
	cfg := l.svcCtx.LLMConfigMgr.Get()

	return &types.LLMConfigResponse{
		APIKey:    l.svcCtx.LLMConfigMgr.MaskAPIKey(),
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		Models:    cfg.Models,
		HasAPIKey: l.svcCtx.LLMConfigMgr.HasAPIKey(),
		UpdatedAt: cfg.UpdatedAt,
	}, nil
}
