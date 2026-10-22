//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPoolDiagnosticStreamRedactsAcrossChunksAndDropsAdminMetadata(t *testing.T) {
	stream := &poolTestStream{model: "public-model", intelligence: true, secrets: poolDiagnosticSecrets(map[string]any{"access_token": "private-split-token"})}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/test", nil).WithContext(context.WithValue(context.Background(), poolTestStreamKey{}, stream))
	svc := &AccountTestService{}
	svc.sendEvent(c, TestEvent{Type: "test_start", Model: "private-model", RequestID: "private-request", Data: "private-data"})
	svc.sendEvent(c, TestEvent{Type: "content", Text: "private-split-"})
	svc.sendEvent(c, TestEvent{Type: "content", Text: "token https://private.example/api person@company.com <svg xmlns=\"http://www.w3.org/2000/svg\">answer</svg>"})
	svc.sendEvent(c, TestEvent{Type: "image", ImageURL: "https://private.example/image", Data: "private-data"})
	require.NotContains(t, recorder.Body.String(), "private-split")
	svc.sendEvent(c, TestEvent{Type: "test_complete", Success: true, UpstreamModel: "private-model", UpstreamStatus: 200})
	raw := recorder.Body.String()
	for _, secret := range []string{"private-split-token", "private.example", "person@company.com", "private-model", "private-data", "private-request", "upstream_status", "image_url"} {
		require.NotContains(t, raw, secret)
	}
	require.Contains(t, raw, "public-model")
	require.Contains(t, raw, "http://www.w3.org/2000/svg")
	require.Contains(t, raw, "answer")
	require.Contains(t, raw, `"success":true`)
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "data: ") {
			var event map[string]any
			require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event))
			for key := range event {
				require.Contains(t, []string{"type", "model", "text", "success"}, key)
			}
		}
	}
}

func TestPoolDiagnosticConnectionAndErrorsNeverReturnRawUpstreamContent(t *testing.T) {
	stream := &poolTestStream{model: "public"}
	require.Empty(t, stream.events(TestEvent{Type: "content", Text: "Authorization: Bearer private"}))
	complete := stream.events(TestEvent{Type: "test_complete", Success: true})
	require.Len(t, complete, 2)
	require.NotContains(t, complete[0].Text, "private")
	failure := stream.events(TestEvent{Type: "error", Error: "proxy private:123 key=private"})
	require.Len(t, failure, 1)
	require.NotContains(t, failure[0].Error, "private")
	stream.intelligence = true
	stream.events(TestEvent{Type: "content", Text: strings.Repeat("x", poolTestOutputLimit+1)})
	require.Equal(t, "error", stream.events(TestEvent{Type: "test_complete", Success: true})[0].Type)
}
