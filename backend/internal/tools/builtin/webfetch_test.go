package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWebFetchTool(t *testing.T) {
	tool := NewWebFetchTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "webfetch", tool.Name())
	assert.Contains(t, tool.Description(), "Fetch content")
}

func TestWebFetchTool_ExecuteMissingURL(t *testing.T) {
	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Contains(t, result, "Error: url is required")
}

func TestWebFetchTool_ExecuteInvalidURL(t *testing.T) {
	tool := NewWebFetchTool()

	tests := []string{
		"ftp://example.com",
		"file:///etc/passwd",
		"example.com",
		"",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]interface{}{"url": url})
			assert.Contains(t, result, "Error:")
		})
	}
}

func TestWebFetchTool_ExecuteHTMLToMarkdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Test</title></head><body><h1>Hello</h1><p>World</p></body></html>`))
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"format": "markdown",
	})

	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "World")
}

func TestWebFetchTool_ExecuteHTMLToText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Title</h1><p>Content</p><script>ignore</script></body></html>`))
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"format": "text",
	})

	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
	assert.NotContains(t, result, "ignore")
}

func TestWebFetchTool_ExecuteHTMLRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Raw HTML</body></html>`))
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"format": "html",
	})

	assert.Contains(t, result, "<html>")
	assert.Contains(t, result, "Raw HTML")
}

func TestWebFetchTool_ExecuteImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG header bytes
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	assert.Contains(t, result, "Image fetched successfully")
	assert.Contains(t, result, "data:image/png;base64,")
}

func TestWebFetchTool_ExecuteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	assert.Contains(t, result, "Error: HTTP 404")
}

func TestWebFetchTool_ExecuteLargeResponse(t *testing.T) {
	largeContent := strings.Repeat("x", 6*1024*1024) // 6MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "6000000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	assert.Contains(t, result, "Error: Response too large")
}

func TestWebFetchTool_ExecuteTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response
		http.Error(w, "slow", http.StatusOK)
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()

	result := tool.Execute(ctx, map[string]interface{}{
		"url": server.URL,
	})

	assert.Contains(t, result, "Error:")
}

func TestWebFetchTool_ExecuteInvalidFormat(t *testing.T) {
	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url":    "https://example.com",
		"format": "json",
	})

	assert.Contains(t, result, "Error: format must be")
}

func TestWebFetchTool_Schema(t *testing.T) {
	tool := NewWebFetchTool()

	schema := tool.Schema()
	assert.Equal(t, "webfetch", schema.Function.Name)
	assert.NotNil(t, schema.Function.Parameters)

	params := schema.Function.Parameters
	assert.Contains(t, params, "properties")
}

func TestWebFetchTool_DefaultFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Test</h1></body></html>`))
	}))
	defer server.Close()

	tool := NewWebFetchTool()

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	assert.Contains(t, result, "Test")
}

func TestBuildAcceptHeader(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"markdown", "text/markdown"},
		{"text", "text/plain"},
		{"html", "text/html"},
		{"unknown", "*/*"},
	}

	for _, tt := range tests {
		result := buildAcceptHeader(tt.format)
		assert.Contains(t, result, tt.expected)
	}
}

func TestIsImage(t *testing.T) {
	tests := []struct {
		mime     string
		expected bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"image/svg+xml", false},
		{"text/html", false},
		{"application/json", false},
	}

	for _, tt := range tests {
		result := isImage(tt.mime)
		assert.Equal(t, tt.expected, result)
	}
}
