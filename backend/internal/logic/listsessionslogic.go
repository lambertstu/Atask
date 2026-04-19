// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"sort"
	"time"

	"agent-base/internal/svc"
	"agent-base/internal/systems/session"
	"agent-base/internal/types"

	"github.com/sashabaranov/go-openai"
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
	var sessions []*session.Session

	if req.ProjectPath != "" {
		project := l.svcCtx.ProjectManager.GetProject(req.ProjectPath)
		if project != nil && len(project.Sessions) > 0 {
			sessions = l.svcCtx.SessionManager.ListSessionsByIDs(project.Sessions)
		}
	}

	if sessions == nil {
		return &types.SessionListResponse{Sessions: nil}, nil
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	var list []types.SessionResponse
	for _, s := range sessions {
		list = append(list, convertSessionToResponse(s))
	}
	return &types.SessionListResponse{Sessions: list}, nil
}

func convertSessionToResponse(s *session.Session) types.SessionResponse {
	return types.SessionResponse{
		ID:          s.ID,
		ProjectPath: s.ProjectPath,
		Model:       s.Model,
		State:       string(s.State),
		Mode:        s.Mode,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		Input:       s.Input,
		Messages:    convertMessages(s.Messages),
		BlockedOn:   s.BlockedOn,
		BlockedTool: s.BlockedTool,
		BlockedArgs: s.BlockedArgs,
	}
}

func convertMessages(msgs []openai.ChatCompletionMessage) []types.ChatMessage {
	if msgs == nil {
		return nil
	}
	result := make([]types.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		cm := types.ChatMessage{
			Role:             string(m.Role),
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			ToolCallID:       m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			cm.ToolCalls = make([]types.ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				cm.ToolCalls = append(cm.ToolCalls, types.ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: types.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		result = append(result, cm)
	}
	return result
}
