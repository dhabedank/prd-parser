package tests

import (
	"testing"

	"github.com/yourusername/prd-parser/internal/output"
)

func TestDefaultOutputConfig(t *testing.T) {
	config := output.DefaultConfig()

	if config.WorkingDir != "." {
		t.Errorf("WorkingDir = %s, want .", config.WorkingDir)
	}
	if config.DryRun {
		t.Error("DryRun should be false by default")
	}
	if !config.IncludeContext {
		t.Error("IncludeContext should be true by default")
	}
	if !config.IncludeTesting {
		t.Error("IncludeTesting should be true by default")
	}
}

func TestJSONAdapterName(t *testing.T) {
	adapter := output.NewJSONAdapter(output.Config{}, "")
	if adapter.Name() != "json" {
		t.Errorf("Name() = %s, want json", adapter.Name())
	}
}

func TestJSONAdapterIsAvailable(t *testing.T) {
	adapter := output.NewJSONAdapter(output.Config{}, "")
	available, err := adapter.IsAvailable()
	if err != nil {
		t.Errorf("IsAvailable() error = %v", err)
	}
	if !available {
		t.Error("JSON adapter should always be available")
	}
}

func TestBeadsAdapterName(t *testing.T) {
	adapter := output.NewBeadsAdapter(output.Config{})
	if adapter.Name() != "beads" {
		t.Errorf("Name() = %s, want beads", adapter.Name())
	}
}

func TestBeadsAdapterIsAvailable(t *testing.T) {
	adapter := output.NewBeadsAdapter(output.Config{})
	available, err := adapter.IsAvailable()
	// Beads adapter may or may not be available depending on system
	// Just verify it doesn't error unexpectedly
	if err != nil {
		t.Errorf("IsAvailable() error = %v", err)
	}
	t.Logf("Beads adapter available: %v", available)
}

func TestOutputConfigCustomValues(t *testing.T) {
	config := output.Config{
		WorkingDir:     "/custom/path",
		DryRun:         true,
		IncludeContext: false,
		IncludeTesting: false,
	}

	if config.WorkingDir != "/custom/path" {
		t.Errorf("WorkingDir = %s, want /custom/path", config.WorkingDir)
	}
	if !config.DryRun {
		t.Error("DryRun should be true")
	}
	if config.IncludeContext {
		t.Error("IncludeContext should be false")
	}
	if config.IncludeTesting {
		t.Error("IncludeTesting should be false")
	}
}

func TestJSONAdapterWithDryRun(t *testing.T) {
	config := output.Config{DryRun: true}
	adapter := output.NewJSONAdapter(config, "test-output.json")

	// Verify adapter was created with config
	if adapter.Name() != "json" {
		t.Errorf("Name() = %s, want json", adapter.Name())
	}
}

func TestBeadsAdapterWithConfig(t *testing.T) {
	config := output.Config{
		WorkingDir:     "/test/dir",
		DryRun:         true,
		IncludeContext: true,
		IncludeTesting: true,
	}
	adapter := output.NewBeadsAdapter(config)

	// Verify adapter was created
	if adapter.Name() != "beads" {
		t.Errorf("Name() = %s, want beads", adapter.Name())
	}
}
