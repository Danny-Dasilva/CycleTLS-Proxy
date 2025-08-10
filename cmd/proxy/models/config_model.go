// config_model.go - Configuration model for Bubble Tea
package models

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigModel represents the configuration editor
type ConfigModel struct {
	inputs   []textinput.Model
	focused  int
	settings map[string]string
	width    int
	height   int
}

// ConfigField represents a configuration field
type ConfigField struct {
	Key         string
	Label       string
	Description string
	Default     string
	Validator   func(string) bool
}

// NewConfigModel creates a new configuration model
func NewConfigModel() ConfigModel {
	fields := []ConfigField{
		{
			Key:         "PORT",
			Label:       "Server Port",
			Description: "Port for the proxy server to listen on",
			Default:     "8080",
			Validator:   validatePort,
		},
		{
			Key:         "LOG_LEVEL",
			Label:       "Log Level",
			Description: "Logging level (debug, info, warn, error)",
			Default:     "info",
			Validator:   validateLogLevel,
		},
		{
			Key:         "DEFAULT_TIMEOUT",
			Label:       "Default Timeout",
			Description: "Default request timeout in seconds",
			Default:     "30",
			Validator:   validateTimeout,
		},
		{
			Key:         "MAX_SESSIONS",
			Label:       "Max Sessions",
			Description: "Maximum number of concurrent sessions",
			Default:     "100",
			Validator:   validateNumber,
		},
	}
	
	inputs := make([]textinput.Model, len(fields))
	settings := make(map[string]string)
	
	for i, field := range fields {
		ti := textinput.New()
		ti.Placeholder = field.Default
		ti.Focus()
		ti.CharLimit = 50
		ti.Width = 30
		
		// Get current value from environment or use default
		currentValue := os.Getenv(field.Key)
		if currentValue == "" {
			currentValue = field.Default
		}
		ti.SetValue(currentValue)
		
		inputs[i] = ti
		settings[field.Key] = currentValue
	}
	
	// Only first input starts focused
	inputs[0].Focus()
	for i := 1; i < len(inputs); i++ {
		inputs[i].Blur()
	}
	
	return ConfigModel{
		inputs:   inputs,
		settings: settings,
		focused:  0,
	}
}

// Init initializes the config model
func (m ConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the config model
func (m ConfigModel) Update(msg tea.Msg) (ConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down", "j":
			m.nextInput()
		case "shift+tab", "up", "k":
			m.prevInput()
		case "enter":
			// Save current values
			m.saveSettings()
		}
	}
	
	// Update the current input
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	
	return m, cmd
}

// View renders the config model
func (m ConfigModel) View() string {
	var content strings.Builder
	
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true).
		Padding(0, 1).
		Width(m.width).
		Align(lipgloss.Center)
	
	header := headerStyle.Render("⚙️ Configuration Settings")
	content.WriteString(header)
	content.WriteString("\n\n")
	
	// Configuration fields
	fields := []ConfigField{
		{"PORT", "Server Port", "Port for the proxy server to listen on", "8080", validatePort},
		{"LOG_LEVEL", "Log Level", "Logging level (debug, info, warn, error)", "info", validateLogLevel},
		{"DEFAULT_TIMEOUT", "Default Timeout", "Default request timeout in seconds", "30", validateTimeout},
		{"MAX_SESSIONS", "Max Sessions", "Maximum number of concurrent sessions", "100", validateNumber},
	}
	
	// Responsive layout
	if m.width < 80 {
		// Compact vertical layout for small terminals
		for i, field := range fields {
			// Field label and description on same line
			labelDescStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B9D")).
				Bold(true)
			
			labelDesc := fmt.Sprintf("%s: %s", field.Label, field.Description)
			content.WriteString(labelDescStyle.Render(labelDesc))
			content.WriteString("\n")
			
			// Input field (smaller)
			inputStyle := lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#61DAFB")).
				Padding(0, 1).
				MarginBottom(1).
				Width(min(m.width-4, 40))
			
			if i == m.focused {
				inputStyle = inputStyle.BorderForeground(lipgloss.Color("#FF6B9D"))
			}
			
			content.WriteString(inputStyle.Render(m.inputs[i].View()))
			content.WriteString("\n")
		}
	} else {
		// Original layout for larger terminals
		for i, field := range fields {
			// Field label
			labelStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B9D")).
				Bold(true)
			
			content.WriteString(labelStyle.Render(field.Label))
			content.WriteString("\n")
			
			// Field description
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6C7B7F")).
				Italic(true)
			
			content.WriteString(descStyle.Render(field.Description))
			content.WriteString("\n")
			
			// Input field
			inputStyle := lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#61DAFB")).
				Padding(0, 1).
				MarginTop(1).
				MarginBottom(2)
			
			if i == m.focused {
				inputStyle = inputStyle.BorderForeground(lipgloss.Color("#FF6B9D"))
			}
			
			content.WriteString(inputStyle.Render(m.inputs[i].View()))
			content.WriteString("\n")
		}
	}
	
	// Current settings display (compact for small terminals)
	if m.width < 80 {
		// Show settings inline
		settingsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98D8C8")).
			Bold(true).
			MarginTop(1)
		
		content.WriteString(settingsStyle.Render("Settings:"))
		
		settingValueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
		
		for key, value := range m.settings {
			content.WriteString(fmt.Sprintf(" %s=%s", 
				lipgloss.NewStyle().Foreground(lipgloss.Color("#61DAFB")).Render(key),
				settingValueStyle.Render(value)))
		}
		content.WriteString("\n")
	} else {
		// Full settings display for larger terminals
		settingsStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#98D8C8")).
			Padding(1, 2).
			MarginTop(2)
		
		settingsContent := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98D8C8")).
			Bold(true).
			Render("Current Settings:")
		
		settingsContent += "\n\n"
		
		for key, value := range m.settings {
			settingsContent += fmt.Sprintf("%s = %s\n", 
				lipgloss.NewStyle().Foreground(lipgloss.Color("#61DAFB")).Render(key),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(value))
		}
		
		content.WriteString(settingsStyle.Render(settingsContent))
	}
	
	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F")).
		Align(lipgloss.Center).
		Width(m.width).
		MarginTop(1)
	
	instructions := "TAB/↑↓ to navigate • ENTER to save • ESC to return"
	content.WriteString(instructionStyle.Render(instructions))
	
	return content.String()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// nextInput moves focus to the next input field
func (m *ConfigModel) nextInput() {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused + 1) % len(m.inputs)
	m.inputs[m.focused].Focus()
}

// prevInput moves focus to the previous input field
func (m *ConfigModel) prevInput() {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
	m.inputs[m.focused].Focus()
}

// saveSettings saves the current input values
func (m *ConfigModel) saveSettings() {
	fields := []string{"PORT", "LOG_LEVEL", "DEFAULT_TIMEOUT", "MAX_SESSIONS"}
	
	for i, key := range fields {
		if i < len(m.inputs) {
			value := m.inputs[i].Value()
			if value != "" {
				m.settings[key] = value
			}
		}
	}
}

// Validation functions
func validatePort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}

func validateLogLevel(value string) bool {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		if strings.ToLower(value) == level {
			return true
		}
	}
	return false
}

func validateTimeout(value string) bool {
	timeout, err := strconv.Atoi(value)
	return err == nil && timeout > 0 && timeout <= 300
}

func validateNumber(value string) bool {
	num, err := strconv.Atoi(value)
	return err == nil && num > 0
}