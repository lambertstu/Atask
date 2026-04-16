package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-base/internal/config"
	"agent-base/internal/svc"
	"agent-base/internal/systems/session"
	"agent-base/pkg/events"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func setupTestSvcCtx(t *testing.T) (*svc.ServiceContext, *session.SessionManager, *events.EventBus) {
	t.Helper()
	tempDir := t.TempDir()
	sm := session.NewSessionManagerLegacy(tempDir)
	eb := events.NewEventBus()
	svcCtx := &svc.ServiceContext{
		Config:         config.Config{WorkDir: tempDir},
		SessionManager: sm,
		EventBus:       eb,
		Engine:         nil,
	}
	return svcCtx, sm, eb
}

func makeRequest(method, path string, body []byte) *http.Request {
	var reader *bytes.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, path, reader)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func withVars(req *http.Request, vars map[string]string) *http.Request {
	return pathvar.WithVars(req, vars)
}

func TestCreateSession(t *testing.T) {
	svcCtx, _, _ := setupTestSvcCtx(t)
	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{"create with default model", map[string]string{"project_path": "/tmp/test"}, http.StatusOK},
		{"create with custom model", map[string]string{"project_path": "/tmp/test", "model": "glm-4"}, http.StatusOK},
		{"missing project_path", map[string]string{"model": "glm-4"}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := makeRequest("POST", "/api/sessions", body)
			rec := httptest.NewRecorder()
			CreateSessionHandler(svcCtx).ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestListSessions(t *testing.T) {
	svcCtx, sm, _ := setupTestSvcCtx(t)
	sm.CreateSession("/tmp/test", "glm-5")
	req := makeRequest("GET", "/api/sessions", nil)
	rec := httptest.NewRecorder()
	ListSessionsHandler(svcCtx).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetSession(t *testing.T) {
	svcCtx, sm, _ := setupTestSvcCtx(t)
	sess := sm.CreateSession("/tmp/test", "glm-5")
	t.Run("existing session", func(t *testing.T) {
		req := makeRequest("GET", "/api/sessions/"+sess.ID, nil)
		req = withVars(req, map[string]string{"id": sess.ID})
		rec := httptest.NewRecorder()
		GetSessionHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("non-existing session", func(t *testing.T) {
		req := makeRequest("GET", "/api/sessions/nonexistent", nil)
		req = withVars(req, map[string]string{"id": "nonexistent"})
		rec := httptest.NewRecorder()
		GetSessionHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestSubmitInput(t *testing.T) {
	svcCtx, sm, _ := setupTestSvcCtx(t)
	sess := sm.CreateSession("/tmp/test", "glm-5")
	t.Run("submit input", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"input": "hello", "mode": "plan"})
		req := makeRequest("POST", "/api/sessions/"+sess.ID+"/input", body)
		req = withVars(req, map[string]string{"id": sess.ID})
		rec := httptest.NewRecorder()
		SubmitInputHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("non-existing session", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"input": "hello"})
		req := makeRequest("POST", "/api/sessions/nonexistent/input", body)
		req = withVars(req, map[string]string{"id": "nonexistent"})
		rec := httptest.NewRecorder()
		SubmitInputHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestApprovePlan(t *testing.T) {
	svcCtx, sm, _ := setupTestSvcCtx(t)
	sess := sm.CreateSession("/tmp/test", "glm-5")
	sm.SubmitInput(sess.ID, "test input", "plan")
	t.Run("approve existing", func(t *testing.T) {
		req := makeRequest("POST", "/api/sessions/"+sess.ID+"/approve", nil)
		req = withVars(req, map[string]string{"id": sess.ID})
		rec := httptest.NewRecorder()
		ApprovePlanHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("approve non-existing", func(t *testing.T) {
		req := makeRequest("POST", "/api/sessions/nonexistent/approve", nil)
		req = withVars(req, map[string]string{"id": "nonexistent"})
		rec := httptest.NewRecorder()
		ApprovePlanHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUnblockSession(t *testing.T) {
	svcCtx, sm, _ := setupTestSvcCtx(t)
	sess := sm.CreateSession("/tmp/test", "glm-5")
	sm.SubmitInput(sess.ID, "test input", "plan")
	sm.Transition(sess.ID, session.StateProcessing)
	sm.SetBlocked(sess.ID, "waiting", "shell", map[string]interface{}{"cmd": "ls"})
	t.Run("unblock with approval", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"response": "ok", "approved": true})
		req := makeRequest("POST", "/api/sessions/"+sess.ID+"/unblock", body)
		req = withVars(req, map[string]string{"id": sess.ID})
		rec := httptest.NewRecorder()
		UnblockSessionHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("unblock non-existing", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"response": "ok", "approved": true})
		req := makeRequest("POST", "/api/sessions/nonexistent/unblock", body)
		req = withVars(req, map[string]string{"id": "nonexistent"})
		rec := httptest.NewRecorder()
		UnblockSessionHandler(svcCtx).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestStreamSessionEvents(t *testing.T) {
	svcCtx, sm, eb := setupTestSvcCtx(t)
	sess := sm.CreateSession("/tmp/test", "glm-5")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req := makeRequest("GET", "/api/sessions/"+sess.ID+"/events", nil)
	req = withVars(req, map[string]string{"id": sess.ID})
	req = req.WithContext(ctx)
	req = withVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		StreamSessionEventsHandler(svcCtx).ServeHTTP(rec, req)
		close(done)
	}()
	eb.Publish(sess.ID, events.NewEvent(events.EventStateChange, sess.ID, map[string]interface{}{"state": "processing"}))
	<-done
	assert.Equal(t, http.StatusOK, rec.Code)
}
