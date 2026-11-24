package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const OpenAICodexStateProbeQuestionKind = "state_probe"
const intelligenceMaxOutputBytes = 2 << 20

var intelligenceTestSlots = make(chan struct{}, 8)

func ValidateIntelligenceConfig(c *PelicanTestConfig) error {
	if c == nil {
		return errors.New("intelligence configuration is required")
	}
	if len(c.Prompt) > 32000 || len(c.RuleName) > 200 || len(c.ModelID) > 100 {
		return errors.New("intelligence input is too long")
	}
	switch c.QuestionKind {
	case "", "pelican", "candy", OpenAICodexStateProbeQuestionKind:
	default:
		return errors.New("unsupported intelligence question")
	}
	if c.ParallelCount == 0 {
		c.ParallelCount = 1
	}
	if c.ParallelCount < 1 || c.ParallelCount > 8 {
		return errors.New("parallel count must be between 1 and 8")
	}
	if c.QuestionKind == OpenAICodexStateProbeQuestionKind && c.ParallelCount != 1 {
		return errors.New("state probe runs once per account")
	}
	if c.ReasoningEffort != "" && normalizePelicanReasoningEffort(c.ReasoningEffort) != c.ReasoningEffort {
		return errors.New("unsupported reasoning effort")
	}
	return nil
}
func validateIntelligencePlan(p *ScheduledTestPlan) error {
	if p.PelicanConfig == nil {
		return nil
	}
	if strings.TrimSpace(p.ModelID) == "" || len(p.ModelID) > 100 {
		return errors.New("intelligence model is required")
	}
	if p.MaxResults <= 0 {
		p.MaxResults = 50
	}
	if p.MaxResults > 200 {
		return errors.New("intelligence history retention must be at most 200")
	}
	if p.AutoRecover {
		return errors.New("intelligence tests do not change account recovery state")
	}
	return ValidateIntelligenceConfig(p.PelicanConfig)
}
func intelligencePrompt(c *PelicanTestConfig) string {
	prompt := strings.TrimSpace(c.Prompt)
	if prompt == "" {
		if c.QuestionKind == "candy" {
			prompt = CandyPrompt
		} else {
			prompt = "创建一个 HTML，内容是 SVG 绘制一个鹈鹕骑自行车的 2D 动画，你不需要任何测试"
		}
	}
	if c.QuestionKind == "candy" {
		return prompt + "\n\n只输出最终整数，不要解释。"
	}
	return prompt + "\n\n" + PelicanDeliveryContract
}

// RunIntelligencePlan preserves ordinary connectivity schedules and opts IQ
// plans into their prompt and reasoning settings. It never applies penalties.
func (s *AccountTestService) RunIntelligencePlan(ctx context.Context, p *ScheduledTestPlan) ([]*ScheduledTestResult, error) {
	if p.PelicanConfig == nil {
		result, err := s.RunTestBackground(ctx, p.AccountID, p.ModelID)
		if err != nil {
			return nil, err
		}
		return []*ScheduledTestResult{result}, nil
	}
	if err := validateIntelligencePlan(p); err != nil {
		return nil, err
	}
	count := p.PelicanConfig.ParallelCount
	results := make([]*ScheduledTestResult, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = s.runIntelligenceOnce(ctx, p.AccountID, p.ModelID, p.PelicanConfig)
		}(i)
	}
	wg.Wait()
	return results, nil
}

type intelligenceRecorder struct {
	*httptest.ResponseRecorder
	cancel   context.CancelFunc
	overflow bool
}

func (w *intelligenceRecorder) Write(p []byte) (int, error) {
	if w.Body.Len()+len(p) > intelligenceMaxOutputBytes {
		w.overflow = true
		w.cancel()
		return 0, errors.New("intelligence output limit reached")
	}
	return w.ResponseRecorder.Write(p)
}
func (s *AccountTestService) runIntelligenceOnce(parent context.Context, id int64, model string, cfg *PelicanTestConfig) *ScheduledTestResult {
	started := time.Now()
	snapshot := *cfg
	snapshot.ModelID = model
	result := &ScheduledTestResult{Status: "failed", StartedAt: started, PelicanConfig: &snapshot}
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	if cfg.QuestionKind == OpenAICodexStateProbeQuestionKind {
		probe, err := s.ProbeOpenAICodexState(ctx, id, model)
		if err != nil {
			result.ErrorMessage = err.Error()
		} else {
			result.ResponseText = fmt.Sprintf("%s\n%s\n%s", probe.Verdict, probe.Reason, OpenAICodexStateProbeLimitation)
			switch probe.Verdict {
			case OpenAICodexStateHealthy:
				result.Status = "success"
			case OpenAICodexStateDegraded:
				result.ErrorMessage = "state_degraded"
			default:
				result.ErrorMessage = "state_probe_inconclusive: " + probe.Failure
			}
		}
	} else {
		w := &intelligenceRecorder{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}
		c, _ := gin.CreateTestContext(w)
		c.Request = (&http.Request{Header: make(http.Header)}).WithContext(ctx)
		c.Set(accountTestBackgroundKey, true)
		err := s.TestPelicanAccountConnection(c, id, model, intelligencePrompt(cfg), cfg.ReasoningEffort)
		result.ResponseText, result.ErrorMessage = parseIntelligenceSSEOutput(w.Body.String())
		if w.overflow {
			result.ErrorMessage = "intelligence output limit reached"
		} else if err != nil && result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
		if result.ErrorMessage == "" {
			result.Status = "success"
		}
	}
	result.FinishedAt = time.Now()
	result.LatencyMs = result.FinishedAt.Sub(started).Milliseconds()
	return result
}

// Require the explicit successful terminator. Partial text is never a pass.
func parseIntelligenceSSEOutput(body string) (string, string) {
	text, failure := parseTestSSEOutput(body)
	complete := false
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event TestEvent
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event) != nil {
			if failure == "" {
				failure = "invalid intelligence event"
			}
			continue
		}
		if event.Type == "test_complete" {
			complete = event.Success
			if !complete && failure == "" {
				failure = "intelligence test failed"
			}
		}
	}
	if !complete && failure == "" {
		failure = "intelligence response did not complete"
	}
	return text, failure
}
