package service

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type poolTestStreamKey struct{}

type poolTestStream struct {
	model        string
	intelligence bool
	output       strings.Builder
	secrets      []string
	truncated    bool
}

const poolTestOutputLimit = 2 << 20

var (
	poolTestEmailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	poolTestURLPattern   = regexp.MustCompile(`https?://[^\s"'<>]+`)
	poolTestTokenPattern = regexp.MustCompile(`(?i)(?:bearer\s+|(?:api[_-]?key|access[_-]?token|refresh[_-]?token)\s*[:=]\s*)[^\s,"'<>]+`)
)

// Pool diagnostics share the admin test implementation, with fixed server-side
// questions and an independent output allowlist. They never perform the admin
// handler's manual recovery/reset operation.
func (s *AccountTestService) TestPoolAccountConnection(c *gin.Context, account *Account, modelID, questionKind, reasoningEffort string) error {
	stream := &poolTestStream{model: modelID, intelligence: questionKind != "", secrets: poolDiagnosticSecrets(account.Credentials)}
	ctx := context.WithValue(c.Request.Context(), poolTestStreamKey{}, stream)
	c.Request = c.Request.WithContext(ctx)
	if questionKind == "" {
		return s.TestAccountConnection(c, account.ID, modelID, "", AccountTestModeDefault)
	}
	prompt := CandyPrompt + "\n\n只输出最终整数，不要解释。"
	if questionKind == "pelican" {
		prompt = "创建一个 HTML，内容是 SVG 绘制一个鹈鹕骑自行车的 2D 动画，你不需要任何测试\n\n" + PelicanDeliveryContract
	}
	return s.TestPelicanAccountConnection(c, account.ID, modelID, prompt, reasoningEffort)
}

func poolDiagnosticSecrets(credentials map[string]any) []string {
	secrets := []string{}
	var collect func(any)
	collect = func(value any) {
		switch v := value.(type) {
		case string:
			if len(v) >= 8 {
				secrets = append(secrets, v)
			}
		case map[string]any:
			for key, item := range v {
				if key != "model_mapping" {
					collect(item)
				}
			}
		case []any:
			for _, item := range v {
				collect(item)
			}
		}
	}
	collect(credentials)
	// Replace longer overlapping secrets first.
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	return secrets
}

func (stream *poolTestStream) safeOutput() string {
	text := stream.output.String()
	for _, secret := range stream.secrets {
		text = strings.ReplaceAll(text, secret, "[redacted]")
	}
	text = poolTestEmailPattern.ReplaceAllString(text, "[redacted]")
	text = poolTestTokenPattern.ReplaceAllString(text, "[redacted]")
	text = poolTestURLPattern.ReplaceAllStringFunc(text, func(url string) string {
		// SVG namespace declarations are part of the built-in animation task.
		if url == "http://www.w3.org/2000/svg" || url == "http://www.w3.org/1999/xlink" {
			return url
		}
		return "[redacted]"
	})
	return strings.ToValidUTF8(text, "")
}

func (stream *poolTestStream) events(event TestEvent) []TestEvent {
	switch event.Type {
	case "test_start":
		return []TestEvent{{Type: "test_start", Model: stream.model}}
	case "content":
		// Buffer before sanitizing: an SSE chunk boundary must not allow a
		// credential split across two tokens to bypass redaction.
		if stream.intelligence {
			remaining := poolTestOutputLimit - stream.output.Len()
			if len(event.Text) > remaining {
				stream.truncated = true
				event.Text = event.Text[:remaining]
			}
			_, _ = stream.output.WriteString(event.Text)
		}
	case "test_complete":
		if stream.truncated {
			return []TestEvent{{Type: "error", Error: "Test output exceeded the size limit"}}
		}
		text := "Connection test completed."
		if stream.intelligence {
			text = stream.safeOutput()
		}
		return []TestEvent{{Type: "content", Text: text}, {Type: "test_complete", Success: event.Success}}
	case "error":
		return []TestEvent{{Type: "error", Error: "测试未完成，请稍后重试；如持续失败，请联系管理员检查账号状态。"}}
	}
	return nil
}
