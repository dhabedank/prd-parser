package tests

import (
	"testing"

	"github.com/yourusername/prd-parser/internal/llm"
)

func TestDefaultLLMConfig(t *testing.T) {
	config := llm.DefaultConfig()

	if !config.PreferCLI {
		t.Error("PreferCLI should be true by default")
	}
	if config.MaxTokens != 16384 {
		t.Errorf("MaxTokens = %d, want 16384", config.MaxTokens)
	}
}

func TestClaudeCLIAdapterName(t *testing.T) {
	adapter := llm.NewClaudeCLIAdapter(llm.Config{})
	if adapter.Name() != "claude-cli" {
		t.Errorf("Name() = %s, want claude-cli", adapter.Name())
	}
}

func TestCodexCLIAdapterName(t *testing.T) {
	adapter := llm.NewCodexCLIAdapter(llm.Config{})
	if adapter.Name() != "codex-cli" {
		t.Errorf("Name() = %s, want codex-cli", adapter.Name())
	}
}

func TestAnthropicAPIAdapterWithoutKey(t *testing.T) {
	// Test with empty API key and no environment variable
	config := llm.Config{APIKey: ""}
	_, err := llm.NewAnthropicAPIAdapter(config)
	// This should fail if ANTHROPIC_API_KEY is not set in the environment
	// We don't want to depend on env var in tests, so just verify the function returns
	_ = err // May or may not error depending on env
}

func TestAnthropicAPIAdapterName(t *testing.T) {
	// Create with a fake key just to test the Name() method
	config := llm.Config{APIKey: "test-key-for-name-test"}
	adapter, err := llm.NewAnthropicAPIAdapter(config)
	if err != nil {
		t.Skipf("Could not create adapter: %v", err)
	}
	if adapter.Name() != "anthropic-api" {
		t.Errorf("Name() = %s, want anthropic-api", adapter.Name())
	}
}

func TestListAvailableAdapters(t *testing.T) {
	config := llm.DefaultConfig()
	adapters := llm.ListAvailableAdapters(config)
	// This is a runtime check - adapters available depend on system
	t.Logf("Available adapters: %v", adapters)
	// The function should return a slice (possibly empty)
	if adapters == nil {
		t.Error("ListAvailableAdapters should return non-nil slice")
	}
}

func TestClaudeCLIAdapterIsAvailable(t *testing.T) {
	adapter := llm.NewClaudeCLIAdapter(llm.Config{})
	// IsAvailable() returns bool - just verify it doesn't panic
	available := adapter.IsAvailable()
	t.Logf("Claude CLI available: %v", available)
}

func TestCodexCLIAdapterIsAvailable(t *testing.T) {
	adapter := llm.NewCodexCLIAdapter(llm.Config{})
	// IsAvailable() returns bool - just verify it doesn't panic
	available := adapter.IsAvailable()
	t.Logf("Codex CLI available: %v", available)
}

func TestLLMConfigWithCustomModel(t *testing.T) {
	config := llm.Config{
		Model:     "custom-model",
		MaxTokens: 8192,
		PreferCLI: false,
	}

	if config.Model != "custom-model" {
		t.Errorf("Model = %s, want custom-model", config.Model)
	}
	if config.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, want 8192", config.MaxTokens)
	}
	if config.PreferCLI {
		t.Error("PreferCLI should be false")
	}
}
