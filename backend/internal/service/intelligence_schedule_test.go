package service

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntelligencePlanRejectsUnsafeAndRetiredPolicy(t *testing.T) {
	for _, cfg := range []*PelicanTestConfig{
		{QuestionKind: "unknown", ParallelCount: 1},
		{QuestionKind: "candy", ParallelCount: 5},
		{QuestionKind: "state_probe", ParallelCount: 2},
		{QuestionKind: "pelican", ReasoningEffort: "invented", ParallelCount: 1},
	} {
		require.Error(t, ValidateIntelligenceConfig(cfg))
	}
	p := &ScheduledTestPlan{ModelID: "model", MaxResults: 50, AutoRecover: true, PelicanConfig: &PelicanTestConfig{QuestionKind: "candy", ParallelCount: 1}}
	require.Error(t, validateIntelligencePlan(p), "IQ observations may never reset account policy")
	p.AutoRecover = false
	require.NoError(t, validateIntelligencePlan(p))
	require.Contains(t, intelligencePrompt(p.PelicanConfig), CandyPrompt)
}
func TestIntelligenceOutputRecorderCancelsOversizedGeneration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &intelligenceRecorder{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}
	_, err := w.Write(make([]byte, intelligenceMaxOutputBytes+1))
	require.Error(t, err)
	require.True(t, w.overflow)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.Zero(t, w.Body.Len())
}
func TestIntelligenceProbeRequiresBothCompleteResponses(t *testing.T) {
	complete := []byte("data: {\"type\":\"response.completed\"}\n\n")
	incomplete := []byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n")
	require.NoError(t, validateIntelligenceProbeResponse(complete))
	require.Error(t, validateIntelligenceProbeResponse(incomplete))
	for _, changed := range []bool{false, true} {
		result := &OpenAICodexStateProbeResult{Verdict: OpenAICodexStateInconclusive}
		calls := 0
		runOpenAICodexStateProbe(context.Background(), result, "", func(state, cookie string) (openAICodexStateShot, error) {
			calls++
			if calls == 1 {
				require.Empty(t, state)
				return openAICodexStateShot{status: 200, state: "first", cookies: []string{"__oailb=exit"}}, nil
			}
			require.Equal(t, "first", state)
			require.Equal(t, "__oailb=exit", cookie)
			next := "first"
			if changed {
				next = "replacement"
			}
			return openAICodexStateShot{status: 200, state: next}, nil
		})
		require.Equal(t, 2, calls)
		expected := OpenAICodexStateHealthy
		if changed {
			expected = OpenAICodexStateDegraded
		}
		require.Equal(t, expected, result.Verdict)
	}
	failed := &OpenAICodexStateProbeResult{Verdict: OpenAICodexStateInconclusive}
	runOpenAICodexStateProbe(context.Background(), failed, "", func(string, string) (openAICodexStateShot, error) {
		return openAICodexStateShot{status: 200, state: "first", streamErr: context.DeadlineExceeded}, nil
	})
	require.Equal(t, OpenAICodexStateInconclusive, failed.Verdict)
	require.Equal(t, OpenAICodexStateFailureStreamError, failed.Failure)
}

func TestIntelligenceSchedulesRequireSuccessfulCompletion(t *testing.T) {
	_, failure := parseIntelligenceSSEOutput("data: {\"type\":\"content\",\"text\":\"partial\"}\n\n")
	require.NotEmpty(t, failure)
	text, failure := parseIntelligenceSSEOutput("data: {\"type\":\"content\",\"text\":\"42\"}\n\ndata: {\"type\":\"test_complete\",\"success\":true}\n\n")
	require.Equal(t, "42", text)
	require.Empty(t, failure)
}
