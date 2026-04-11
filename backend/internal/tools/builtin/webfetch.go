package builtin

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/sashabaranov/go-openai"
)

const (
	defaultTimeout  = 30 * time.Second
	maxTimeout      = 120 * time.Second
	maxResponseSize = 5 * 1024 * 1024
)

type WebFetchTool struct {
	timeout   time.Duration
	maxSize   int64
	converter *md.Converter
}

func NewWebFetchTool() *WebFetchTool {
	converter := md.NewConverter("", true, nil)
	return &WebFetchTool{
		timeout:   defaultTimeout,
		maxSize:   maxResponseSize,
		converter: converter,
	}
}

func (t *WebFetchTool) Name() string {
	return "webfetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL and return it in specified format (markdown, text, or html)."
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) string {
	url := utils.GetStringFromMap(args, "url")
	if url == "" {
		return "Error: url is required"
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "Error: URL must start with http:// or https://"
	}

	format := utils.GetStringFromMap(args, "format")
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "text" && format != "html" {
		return "Error: format must be markdown, text, or html"
	}

	timeoutSec := utils.GetIntFromMap(args, "timeout")
	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = t.timeout
	}
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	acceptHeader := buildAcceptHeader(format)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "Error: Request timed out"
		}
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Error: HTTP %d - %s", resp.StatusCode, resp.Status)
	}

	contentLength := resp.ContentLength
	if contentLength > t.maxSize {
		return "Error: Response too large (exceeds 5MB limit)"
	}

	contentType := resp.Header.Get("Content-Type")
	mime := strings.Split(contentType, ";")[0]
	mime = strings.TrimSpace(strings.ToLower(mime))

	if isImage(mime) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		if len(body) > int(t.maxSize) {
			return "Error: Response too large (exceeds 5MB limit)"
		}
		base64Content := base64.StdEncoding.EncodeToString(body)
		return fmt.Sprintf("Image fetched successfully (type: %s, size: %d bytes)\ndata:%s;base64,%s", mime, len(body), mime, base64Content)
	}

	limitedReader := io.LimitReader(resp.Body, t.maxSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if len(body) > int(t.maxSize) {
		return "Error: Response too large (exceeds 5MB limit)"
	}

	content := string(body)

	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml+xml") {
		switch format {
		case "markdown":
			markdown, err := t.converter.ConvertString(content)
			if err != nil {
				return fmt.Sprintf("Error converting to markdown: %v", err)
			}
			return markdown
		case "text":
			text := extractTextFromHTML(content)
			return text
		case "html":
			return content
		}
	}

	return content
}

func (t *WebFetchTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch content from",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"markdown", "text", "html"},
						"default":     "markdown",
						"description": "The format to return the content in",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Optional timeout in seconds (max 120)",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

func RegisterWebFetchTool(registry tools.ToolRegistry) {
	registry.Register(NewWebFetchTool())
}

func buildAcceptHeader(format string) string {
	switch format {
	case "markdown":
		return "text/markdown;q=1.0, text/x-markdown;q=0.9, text/plain;q=0.8, text/html;q=0.7, */*;q=0.1"
	case "text":
		return "text/plain;q=1.0, text/markdown;q=0.9, text/html;q=0.8, */*;q=0.1"
	case "html":
		return "text/html;q=1.0, application/xhtml+xml;q=0.9, text/plain;q=0.8, text/markdown;q=0.7, */*;q=0.1"
	default:
		return "*/*"
	}
}

func isImage(mime string) bool {
	if mime == "image/svg+xml" || mime == "image/vnd.fastbidsheet" {
		return false
	}
	return strings.HasPrefix(mime, "image/")
}

func extractTextFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	doc.Find("script, style, noscript, iframe, object, embed").Remove()

	text := doc.Text()
	return strings.TrimSpace(text)
}
