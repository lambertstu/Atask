// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/rest/httpx"

	"agent-base/internal/logic"
	"agent-base/internal/svc"
	"agent-base/internal/types"
)

func StreamSessionEventsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.StreamSessionEventsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		client := make(chan *types.SessionEvent, 16)

		l := logic.NewStreamSessionEventsLogic(r.Context(), svcCtx)
		threading.GoSafeCtx(r.Context(), func() {
			defer close(client)
			err := l.StreamSessionEvents(&req, client)
			if err != nil {
				logc.Errorw(r.Context(), "StreamSessionEventsHandler", logc.Field("error", err))
				return
			}
		})

		for {
			select {
			case data, ok := <-client:
				if !ok {
					return
				}
				output, err := json.Marshal(data)
				if err != nil {
					logc.Errorw(r.Context(), "StreamSessionEventsHandler", logc.Field("error", err))
					continue
				}

				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(output)); err != nil {
					logc.Errorw(r.Context(), "StreamSessionEventsHandler", logc.Field("error", err))
					return
				}
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	}
}
