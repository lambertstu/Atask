package handler

import (
	"net/http"

	"agent-base/internal/logic"
	"agent-base/internal/svc"
	"agent-base/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ApprovePlanHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ApprovePlanRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewApprovePlanLogic(r.Context(), svcCtx)
		resp, err := l.ApprovePlan(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
