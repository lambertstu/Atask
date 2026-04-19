// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"agent-base/internal/svc"
	"agent-base/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLLMConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLLMConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLLMConfigLogic {
	return &UpdateLLMConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLLMConfigLogic) UpdateLLMConfig(req *types.LLMConfigRequest) (resp *types.LLMConfigResponse, err error) {
	if err := l.svcCtx.LLMConfigMgr.Update(req.APIKey, req.BaseURL, req.Model); err != nil {
		return nil, err
	}

	cfg := l.svcCtx.LLMConfigMgr.Get()

	if l.svcCtx.LLMClient != nil {
		l.svcCtx.LLMClient.UpdateConfig(cfg.APIKey, cfg.BaseURL, cfg.Model)
	}

	return &types.LLMConfigResponse{
		APIKey:    l.svcCtx.LLMConfigMgr.MaskAPIKey(),
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		HasAPIKey: l.svcCtx.LLMConfigMgr.HasAPIKey(),
		UpdatedAt: cfg.UpdatedAt,
	}, nil
}
