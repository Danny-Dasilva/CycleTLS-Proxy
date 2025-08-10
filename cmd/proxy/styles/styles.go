// styles.go - Centralized styling system for CycleTLS-Proxy TUI
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary gradient colors (blue to cyan)
	PrimaryStart = lipgloss.Color("#4F46E5") // Indigo
	PrimaryEnd   = lipgloss.Color("#06B6D4") // Cyan
	
	// Accent colors
	AccentGreen  = lipgloss.Color("#10B981") // Emerald
	AccentOrange = lipgloss.Color("#F59E0B") // Amber
	AccentRed    = lipgloss.Color("#EF4444") // Red
	AccentPurple = lipgloss.Color("#8B5CF6") // Violet
	
	// UI colors
	TextPrimary   = lipgloss.Color("#F9FAFB") // Gray-50
	TextSecondary = lipgloss.Color("#9CA3AF") // Gray-400
	TextMuted     = lipgloss.Color("#6B7280") // Gray-500
	TextDisabled  = lipgloss.Color("#4B5563") // Gray-600
	
	// Background colors
	BgPrimary   = lipgloss.Color("#111827") // Gray-900
	BgSecondary = lipgloss.Color("#1F2937") // Gray-800
	BgTertiary  = lipgloss.Color("#374151") // Gray-700
	
	// Border colors
	BorderFocused   = lipgloss.Color("#06B6D4") // Cyan
	BorderUnfocused = lipgloss.Color("#4B5563") // Gray-600
	BorderMuted     = lipgloss.Color("#374151") // Gray-700
)

// Base styles
var (
	// Title styles with gradient effect
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryEnd).
			Align(lipgloss.Center)
	
	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentGreen).
			MarginBottom(1)
	
	// Content styles
	ContentStyle = lipgloss.NewStyle().
			Foreground(TextPrimary).
			MarginLeft(2)
	
	// Code/command styles
	CodeStyle = lipgloss.NewStyle().
			Foreground(AccentOrange).
			Background(BgSecondary).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)
	
	// Focus styles
	FocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderFocused).
			Padding(1)
	
	UnfocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderUnfocused).
			Padding(1)
	
	// List item styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(PrimaryStart).
				Padding(0, 1)
	
	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(TextSecondary).
				Padding(0, 1)
	
	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(AccentGreen).
			Bold(true)
	
	ErrorStyle = lipgloss.NewStyle().
			Foreground(AccentRed).
			Bold(true)
	
	WarningStyle = lipgloss.NewStyle().
			Foreground(AccentOrange).
			Bold(true)
	
	InfoStyle = lipgloss.NewStyle().
			Foreground(PrimaryEnd).
			Bold(true)
)

// Responsive styles that adapt to terminal size
func GetResponsiveStyle(width, height int) lipgloss.Style {
	if width < 100 {
		// Small terminal style
		return lipgloss.NewStyle().
			Width(width - 4).
			Height(height - 4).
			Padding(1)
	} else {
		// Large terminal style
		return lipgloss.NewStyle().
			Width(width - 8).
			Height(height - 6).
			Padding(2)
	}
}

// Panel styles for split-pane layout
func GetLeftPanelStyle(width, height int, focused bool) lipgloss.Style {
	panelWidth := width/2 - 2
	
	style := lipgloss.NewStyle().
		Width(panelWidth).
		Height(height - 4).
		Padding(1)
	
	if focused {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderFocused)
	} else {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderUnfocused)
	}
	
	return style
}

func GetRightPanelStyle(width, height int, focused bool) lipgloss.Style {
	panelWidth := width/2 - 2
	
	style := lipgloss.NewStyle().
		Width(panelWidth).
		Height(height - 4).
		Padding(1)
	
	if focused {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderFocused)
	} else {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderUnfocused)
	}
	
	return style
}

// Key binding style
func KeyStyle(key string) string {
	return lipgloss.NewStyle().
		Foreground(PrimaryEnd).
		Bold(true).
		Background(BgSecondary).
		Padding(0, 1).
		MarginRight(1).
		Render(" " + key + " ")
}

// Status bar style
func StatusBarStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMuted).
		Background(BgSecondary).
		Width(width).
		Align(lipgloss.Center).
		Padding(0, 1)
}