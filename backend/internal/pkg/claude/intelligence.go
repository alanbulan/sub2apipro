package claude

import "strings"

// SupportsAdaptiveThinking is used by Pro IQ requests only.
func SupportsAdaptiveThinking(model string) bool {
	id := normalizeEffortModelID(model)
	if id == "claude-opus-4-5" || strings.HasPrefix(id, "claude-opus-4-5-") {
		return false
	}
	return EffortLevelsForModel(model) != nil
}
