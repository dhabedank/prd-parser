package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhabedank/prd-parser/internal/llm"
	"github.com/dhabedank/prd-parser/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var resetConfig bool

// SetupCmd represents the setup command.
var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive configuration wizard",
	Long: `Configure prd-parser with an interactive wizard.

This wizard helps you select models for each parsing stage:
- Epic model: Used for generating epics from PRD (Stage 1)
- Task model: Used for generating tasks from epics (Stage 2)
- Subtask model: Used for generating subtasks from tasks (Stage 3)

Configuration is saved to ~/.prd-parser.yaml`,
	RunE: runSetup,
}

func init() {
	SetupCmd.Flags().BoolVar(&resetConfig, "reset", false, "Reset configuration to defaults")
}

// setupConfig holds the configuration being built.
type setupConfig struct {
	EpicModel    string `yaml:"epic_model,omitempty"`
	TaskModel    string `yaml:"task_model,omitempty"`
	SubtaskModel string `yaml:"subtask_model,omitempty"`
}

func runSetup(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Handle reset
	if resetConfig {
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config: %w", err)
		}
		fmt.Println(tui.SuccessStyle.Render("✓") + " Configuration reset to defaults")
		fmt.Printf("  Removed: %s\n", configPath)
		return nil
	}

	// Check for available models
	models := llm.AllModels()
	if len(models) == 0 {
		return fmt.Errorf("no LLM providers detected. Please install Claude Code or Codex CLI")
	}

	// Run the wizard
	p := tea.NewProgram(newSetupModel(models))
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	// Get the final model
	finalModel := m.(setupModel)
	if finalModel.cancelled {
		fmt.Println("Setup cancelled")
		return nil
	}

	// Save configuration
	config := setupConfig{
		EpicModel:    finalModel.selectedModels[0],
		TaskModel:    finalModel.selectedModels[1],
		SubtaskModel: finalModel.selectedModels[2],
	}

	if err := saveConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓") + " Configuration saved to " + configPath)
	fmt.Println()
	fmt.Println("Selected models:")
	fmt.Printf("  Epic:    %s\n", tui.ModelStyle.Render(config.EpicModel))
	fmt.Printf("  Task:    %s\n", tui.ModelStyle.Render(config.TaskModel))
	fmt.Printf("  Subtask: %s\n", tui.ModelStyle.Render(config.SubtaskModel))

	return nil
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".prd-parser.yaml"
	}
	return filepath.Join(home, ".prd-parser.yaml")
}

func saveConfig(path string, config setupConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Bubble Tea model for the setup wizard

type setupModel struct {
	step           int // 0=epic, 1=task, 2=subtask
	lists          []list.Model
	selectedModels []string
	cancelled      bool
	width          int
	height         int
}

type modelItem struct {
	info llm.ModelInfo
}

func (m modelItem) Title() string       { return m.info.Name }
func (m modelItem) Description() string { return m.info.Description }
func (m modelItem) FilterValue() string { return m.info.Name }

func newSetupModel(models []llm.ModelInfo) setupModel {
	items := make([]list.Item, len(models))
	for i, m := range models {
		items[i] = modelItem{info: m}
	}

	// Create three lists (one per step)
	lists := make([]list.Model, 3)
	titles := []string{
		"Select Epic Model (Stage 1)",
		"Select Task Model (Stage 2)",
		"Select Subtask Model (Stage 3)",
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#9b59b6"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#95a5a6"))

	for i := 0; i < 3; i++ {
		l := list.New(items, delegate, 60, 14)
		l.Title = titles[i]
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(false)
		l.Styles.Title = tui.TitleStyle
		lists[i] = l
	}

	return setupModel{
		step:           0,
		lists:          lists,
		selectedModels: make([]string, 3),
	}
}

func (m setupModel) Init() tea.Cmd {
	return nil
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		for i := range m.lists {
			m.lists[i].SetWidth(msg.Width)
			m.lists[i].SetHeight(msg.Height - 4)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			// Select current item
			if item, ok := m.lists[m.step].SelectedItem().(modelItem); ok {
				m.selectedModels[m.step] = item.info.ID
			}

			// Move to next step or finish
			m.step++
			if m.step >= 3 {
				return m, tea.Quit
			}
			return m, nil

		case "left", "h":
			if m.step > 0 {
				m.step--
			}
			return m, nil
		}
	}

	// Update current list
	var cmd tea.Cmd
	m.lists[m.step], cmd = m.lists[m.step].Update(msg)
	return m, cmd
}

func (m setupModel) View() string {
	if m.cancelled {
		return ""
	}

	// Progress indicator
	steps := []string{"Epic", "Task", "Subtask"}
	progress := "\n  "
	for i, s := range steps {
		if i == m.step {
			progress += tui.SelectedStyle.Render(fmt.Sprintf("[%s]", s))
		} else if i < m.step {
			progress += tui.SuccessStyle.Render(fmt.Sprintf("✓ %s", s))
		} else {
			progress += tui.UnselectedStyle.Render(fmt.Sprintf("○ %s", s))
		}
		if i < len(steps)-1 {
			progress += " → "
		}
	}
	progress += "\n\n"

	// Help text
	help := tui.HelpStyle.Render("\n  ↑/↓: navigate • enter: select • ←: back • q: quit")

	return progress + m.lists[m.step].View() + help
}
