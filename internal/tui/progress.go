package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// StageInfo holds information about a parsing stage.
type StageInfo struct {
	Name        string
	Model       string
	InputChars  int
	StartTime   time.Time
	EndTime     time.Time
	IsComplete  bool
	OutputChars int
}

// ProgressDisplay is a Bubble Tea model for showing parsing progress.
type ProgressDisplay struct {
	spinner     spinner.Model
	stages      []StageInfo
	currentIdx  int
	isRunning   bool
	totalCost   float64
	quitting    bool
}

// NewProgressDisplay creates a new progress display.
func NewProgressDisplay() *ProgressDisplay {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return &ProgressDisplay{
		spinner:    s,
		stages:     []StageInfo{},
		currentIdx: -1,
		isRunning:  false,
	}
}

// StartStage begins tracking a new stage.
func (p *ProgressDisplay) StartStage(name, model string, inputChars int) {
	stage := StageInfo{
		Name:       name,
		Model:      model,
		InputChars: inputChars,
		StartTime:  time.Now(),
	}
	p.stages = append(p.stages, stage)
	p.currentIdx = len(p.stages) - 1
	p.isRunning = true
}

// CompleteStage marks the current stage as complete.
func (p *ProgressDisplay) CompleteStage(outputChars int) {
	if p.currentIdx >= 0 && p.currentIdx < len(p.stages) {
		p.stages[p.currentIdx].IsComplete = true
		p.stages[p.currentIdx].EndTime = time.Now()
		p.stages[p.currentIdx].OutputChars = outputChars

		// Calculate cost for this stage
		inputTokens := EstimateTokens(p.stages[p.currentIdx].InputChars)
		outputTokens := EstimateTokens(outputChars)
		cost := EstimateCost(p.stages[p.currentIdx].Model, inputTokens, outputTokens)
		p.totalCost += cost
	}
	p.isRunning = false
}

// Stop stops the progress display.
func (p *ProgressDisplay) Stop() {
	p.isRunning = false
	p.quitting = true
}

// Init implements tea.Model.
func (p *ProgressDisplay) Init() tea.Cmd {
	return p.spinner.Tick
}

// Update implements tea.Model.
func (p *ProgressDisplay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			p.quitting = true
			return p, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd
	}

	return p, nil
}

// View implements tea.Model.
func (p *ProgressDisplay) View() string {
	if p.quitting {
		return p.summaryView()
	}

	if p.currentIdx < 0 || p.currentIdx >= len(p.stages) {
		return ""
	}

	stage := p.stages[p.currentIdx]
	elapsed := time.Since(stage.StartTime).Truncate(time.Second)
	inputTokens := EstimateTokens(stage.InputChars)

	// Build the progress line
	var status string
	if p.isRunning {
		status = p.spinner.View()
	} else {
		status = SuccessStyle.Render("✓")
	}

	line := fmt.Sprintf("%s %s  %s  %s  ~%s input",
		status,
		StageStyle.Render(stage.Name),
		ModelStyle.Render(stage.Model),
		HelpStyle.Render(elapsed.String()),
		FormatTokens(inputTokens),
	)

	return line
}

// summaryView shows the final summary after completion.
func (p *ProgressDisplay) summaryView() string {
	if len(p.stages) == 0 {
		return ""
	}

	var totalInputTokens, totalOutputTokens int
	var totalDuration time.Duration

	for _, stage := range p.stages {
		totalInputTokens += EstimateTokens(stage.InputChars)
		totalOutputTokens += EstimateTokens(stage.OutputChars)
		if stage.IsComplete {
			totalDuration += stage.EndTime.Sub(stage.StartTime)
		}
	}

	return fmt.Sprintf("\n%s\n  Stages: %d  Tokens: ~%s in / ~%s out  Cost: %s  Time: %s\n",
		TitleStyle.Render("Generation Complete"),
		len(p.stages),
		FormatTokens(totalInputTokens),
		FormatTokens(totalOutputTokens),
		CostStyle.Render(FormatCost(p.totalCost)),
		totalDuration.Truncate(time.Second).String(),
	)
}

// RenderStageStart returns a string for stage start (non-interactive mode).
func RenderStageStart(name, model string, inputChars int) string {
	inputTokens := EstimateTokens(inputChars)
	return fmt.Sprintf("%s %s  %s  ~%s input tokens",
		SpinnerStyle.Render("→"),
		StageStyle.Render(name),
		ModelStyle.Render(model),
		FormatTokens(inputTokens),
	)
}

// RenderStageComplete returns a string for stage completion (non-interactive mode).
func RenderStageComplete(name string, duration time.Duration, inputChars, outputChars int, model string) string {
	inputTokens := EstimateTokens(inputChars)
	outputTokens := EstimateTokens(outputChars)
	cost := EstimateCost(model, inputTokens, outputTokens)

	return fmt.Sprintf("%s %s  %s  ~%s tokens  %s",
		SuccessStyle.Render("✓"),
		StageStyle.Render(name),
		HelpStyle.Render(duration.Truncate(time.Second).String()),
		FormatTokens(inputTokens+outputTokens),
		CostStyle.Render(FormatCost(cost)),
	)
}

// RenderSummary returns a summary string (non-interactive mode).
func RenderSummary(stages []StageInfo) string {
	var totalInputTokens, totalOutputTokens int
	var totalCost float64
	var totalDuration time.Duration

	for _, stage := range stages {
		inputTokens := EstimateTokens(stage.InputChars)
		outputTokens := EstimateTokens(stage.OutputChars)
		totalInputTokens += inputTokens
		totalOutputTokens += outputTokens
		totalCost += EstimateCost(stage.Model, inputTokens, outputTokens)
		if stage.IsComplete {
			totalDuration += stage.EndTime.Sub(stage.StartTime)
		}
	}

	return fmt.Sprintf("\n%s\n  Stages: %d  Tokens: ~%s in / ~%s out  Est. cost: %s  Time: %s\n",
		TitleStyle.Render("Generation Complete"),
		len(stages),
		FormatTokens(totalInputTokens),
		FormatTokens(totalOutputTokens),
		CostStyle.Render(FormatCost(totalCost)),
		totalDuration.Truncate(time.Second).String(),
	)
}
