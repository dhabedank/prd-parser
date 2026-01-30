package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// ModelInfo describes an available model.
type ModelInfo struct {
	ID          string // Model identifier (e.g., "claude-opus-4-5-20251101")
	Name        string // Human-readable name (e.g., "Claude Opus 4.5")
	Description string // Brief description
	Provider    string // Provider name (e.g., "anthropic", "openai")
}

// claudeModels lists Claude models available via CLI.
// Updated: 2026-01-30 from https://docs.anthropic.com/en/docs/about-claude/models
var claudeModels = []ModelInfo{
	// Latest 4.5 models
	{ID: "claude-opus-4-5-20251101", Name: "Claude Opus 4.5", Description: "Premium model, maximum intelligence ($5/$25 per MTok)", Provider: "anthropic"},
	{ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5", Description: "Best balance of speed and capability ($3/$15 per MTok)", Provider: "anthropic"},
	{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", Description: "Fastest, most cost-effective ($1/$5 per MTok)", Provider: "anthropic"},
	// Legacy models
	{ID: "claude-opus-4-1-20250805", Name: "Claude Opus 4.1", Description: "Previous premium model ($15/$75 per MTok)", Provider: "anthropic"},
	{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Description: "Previous balanced model ($3/$15 per MTok)", Provider: "anthropic"},
	{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", Description: "Legacy premium ($15/$75 per MTok)", Provider: "anthropic"},
	{ID: "claude-3-7-sonnet-20250219", Name: "Claude 3.7 Sonnet", Description: "Legacy fast model ($3/$15 per MTok)", Provider: "anthropic"},
	{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Description: "Legacy budget model ($0.25/$1.25 per MTok)", Provider: "anthropic"},
}

// codexModels lists Codex/OpenAI models available via CLI.
var codexModels = []ModelInfo{
	{ID: "o3", Name: "O3", Description: "Most capable reasoning model", Provider: "openai"},
	{ID: "o3-mini", Name: "O3 Mini", Description: "Fast reasoning model", Provider: "openai"},
	{ID: "o1", Name: "O1", Description: "Advanced reasoning", Provider: "openai"},
	{ID: "o1-mini", Name: "O1 Mini", Description: "Efficient reasoning", Provider: "openai"},
	{ID: "gpt-4o", Name: "GPT-4o", Description: "Fast multimodal model", Provider: "openai"},
	{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Most cost-effective", Provider: "openai"},
}

// AvailableModels returns models grouped by provider based on available CLIs.
// If ANTHROPIC_API_KEY is set, it will fetch the latest models from the API.
func AvailableModels() map[string][]ModelInfo {
	result := make(map[string][]ModelInfo)

	// Check for Claude CLI
	if _, err := exec.LookPath("claude"); err == nil {
		// Try to fetch models from API if key is available
		if models := fetchAnthropicModels(); len(models) > 0 {
			result["anthropic"] = models
		} else {
			result["anthropic"] = claudeModels
		}
	}

	// Check for Codex CLI
	if _, err := exec.LookPath("codex"); err == nil {
		// Try to fetch models from OpenAI API if key is available
		if models := fetchOpenAIModels(); len(models) > 0 {
			result["openai"] = models
		} else {
			result["openai"] = codexModels
		}
	}

	return result
}

// fetchAnthropicModels fetches available models from the Anthropic API.
// Returns nil if API key is not set or request fails.
func fetchAnthropicModels() []ModelInfo {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var response struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil
	}

	var models []ModelInfo
	for _, m := range response.Data {
		if m.Type != "model" {
			continue
		}
		// Parse model name for description
		desc := inferModelDescription(m.ID)
		name := m.DisplayName
		if name == "" {
			name = formatModelName(m.ID)
		}
		models = append(models, ModelInfo{
			ID:          m.ID,
			Name:        name,
			Description: desc,
			Provider:    "anthropic",
		})
	}

	// Sort by model ID (newer models first based on date suffix)
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID > models[j].ID
	})

	return models
}

// fetchOpenAIModels fetches available models from the OpenAI API.
// Returns nil if API key is not set or request fails.
func fetchOpenAIModels() []ModelInfo {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil
	}

	// Filter to relevant models (gpt-4, o1, o3, etc.)
	relevantPrefixes := []string{"gpt-4", "gpt-5", "o1", "o3"}
	var models []ModelInfo
	for _, m := range response.Data {
		isRelevant := false
		for _, prefix := range relevantPrefixes {
			if strings.HasPrefix(m.ID, prefix) {
				isRelevant = true
				break
			}
		}
		if !isRelevant {
			continue
		}

		models = append(models, ModelInfo{
			ID:          m.ID,
			Name:        formatModelName(m.ID),
			Description: inferModelDescription(m.ID),
			Provider:    "openai",
		})
	}

	// Sort by model ID
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID > models[j].ID
	})

	return models
}

// formatModelName converts a model ID to a human-readable name.
func formatModelName(id string) string {
	// Handle Claude models
	if strings.HasPrefix(id, "claude-") {
		parts := strings.Split(id, "-")
		if len(parts) >= 3 {
			// e.g., "claude-sonnet-4-5-20250929" -> "Claude Sonnet 4.5"
			name := "Claude"
			for i := 1; i < len(parts)-1; i++ {
				// Skip date suffix
				if len(parts[i]) == 8 && parts[i][0] == '2' {
					break
				}
				name += " " + strings.Title(parts[i])
			}
			return name
		}
	}

	// Handle OpenAI models
	return strings.ToUpper(id[:1]) + id[1:]
}

// inferModelDescription returns a description based on model ID patterns.
func inferModelDescription(id string) string {
	id = strings.ToLower(id)

	// Claude models
	if strings.Contains(id, "opus") {
		if strings.Contains(id, "4-5") || strings.Contains(id, "4.5") {
			return "Premium model, maximum intelligence ($5/$25 per MTok)"
		}
		return "Premium model ($15/$75 per MTok)"
	}
	if strings.Contains(id, "sonnet") {
		if strings.Contains(id, "4-5") || strings.Contains(id, "4.5") {
			return "Best balance of speed and capability ($3/$15 per MTok)"
		}
		return "Balanced speed and capability ($3/$15 per MTok)"
	}
	if strings.Contains(id, "haiku") {
		if strings.Contains(id, "4-5") || strings.Contains(id, "4.5") {
			return "Fastest, most cost-effective ($1/$5 per MTok)"
		}
		return "Fast and cost-effective"
	}

	// OpenAI models
	if strings.HasPrefix(id, "o3") {
		if strings.Contains(id, "mini") {
			return "Fast reasoning model"
		}
		return "Most capable reasoning model"
	}
	if strings.HasPrefix(id, "o1") {
		if strings.Contains(id, "mini") {
			return "Efficient reasoning"
		}
		return "Advanced reasoning"
	}
	if strings.Contains(id, "gpt-4o") {
		if strings.Contains(id, "mini") {
			return "Most cost-effective"
		}
		return "Fast multimodal model"
	}

	return ""
}

// AllModels returns a flat list of all available models.
func AllModels() []ModelInfo {
	available := AvailableModels()
	var result []ModelInfo

	// Add Claude models first (preferred)
	if models, ok := available["anthropic"]; ok {
		result = append(result, models...)
	}

	// Add OpenAI models
	if models, ok := available["openai"]; ok {
		result = append(result, models...)
	}

	return result
}

// DetectBestAdapter finds the best available LLM adapter.
// Priority: Claude CLI > Codex CLI > Anthropic API
func DetectBestAdapter(config Config) (Adapter, error) {
	// Try Claude CLI first (preferred - already authenticated)
	if config.PreferCLI {
		claude := NewClaudeCLIAdapter(config)
		if claude.IsAvailable() {
			return claude, nil
		}

		// Try Codex CLI
		codex := NewCodexCLIAdapter(config)
		if codex.IsAvailable() {
			return codex, nil
		}
	}

	// Fall back to Anthropic API
	anthropic, err := NewAnthropicAPIAdapter(config)
	if err == nil && anthropic.IsAvailable() {
		return anthropic, nil
	}

	// Could add OpenAI API fallback here

	return nil, fmt.Errorf("no LLM adapter available - install Claude Code, Codex, or set ANTHROPIC_API_KEY")
}

// ListAvailableAdapters returns all adapters that could be used.
func ListAvailableAdapters(config Config) []string {
	available := []string{}

	claude := NewClaudeCLIAdapter(config)
	if claude.IsAvailable() {
		available = append(available, "claude-cli")
	}

	codex := NewCodexCLIAdapter(config)
	if codex.IsAvailable() {
		available = append(available, "codex-cli")
	}

	anthropic, _ := NewAnthropicAPIAdapter(config)
	if anthropic != nil && anthropic.IsAvailable() {
		available = append(available, "anthropic-api")
	}

	return available
}
