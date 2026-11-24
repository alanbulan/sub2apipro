package service

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPelicanPayloadsDoNotChangeDefaultAccountTestPayloads(t *testing.T) {
	defaultClaude, err := createTestPayload("claude-sonnet-4-6")
	require.NoError(t, err)
	defaultMessages, ok := defaultClaude["messages"].([]map[string]any)
	require.True(t, ok)
	defaultContent, ok := defaultMessages[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "hi", defaultContent[0]["text"])
	require.Equal(t, 1024, defaultClaude["max_tokens"])
	require.NotContains(t, defaultClaude, "output_config")
	require.NotContains(t, defaultClaude, "thinking")

	defaultOpenAI := createOpenAITestPayload("gpt-6-astra", true)
	defaultInput, ok := defaultOpenAI["input"].([]map[string]any)
	require.True(t, ok)
	defaultOpenAIContent, ok := defaultInput[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "hi", defaultOpenAIContent[0]["text"])
	require.NotContains(t, defaultOpenAI, "reasoning")

	pelicanClaude, err := createPelicanClaudePayload("claude-sonnet-4-6", "draw the pelican animation", "")
	require.NoError(t, err)
	pelicanMessages, ok := pelicanClaude["messages"].([]map[string]any)
	require.True(t, ok)
	pelicanContent, ok := pelicanMessages[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "draw the pelican animation", pelicanContent[0]["text"])

	pelicanOpenAI := createPelicanOpenAIPayload("gpt-6-astra", true, "draw the pelican animation", "medium")
	pelicanInput, ok := pelicanOpenAI["input"].([]map[string]any)
	require.True(t, ok)
	pelicanOpenAIContent, ok := pelicanInput[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "draw the pelican animation", pelicanOpenAIContent[0]["text"])
	require.Equal(t, map[string]any{"effort": "medium"}, pelicanOpenAI["reasoning"])
}

func TestIntelligenceClaudeStreamRejectsTruncatedAndRefusedAnswers(t *testing.T) {
	for _, body := range []string{
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"partial\"}}\n\n",
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"max_tokens\"}}\n\ndata: {\"type\":\"message_stop\"}\n\n",
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"refusal\"}}\n\ndata: {\"type\":\"message_stop\"}\n\n",
	} {
		writer := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(writer)
		c.Request = httptest.NewRequest("POST", "/iq", nil).WithContext(withPelicanTestOptions(context.Background(), pelicanTestOptions{}))
		svc := &AccountTestService{}
		require.Error(t, svc.processClaudeStream(c, strings.NewReader(body)))
		require.NotContains(t, writer.Body.String(), `"success":true`)
	}
}
