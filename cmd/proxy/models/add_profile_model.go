// add_profile_model.go - Add New Profile form model for Bubble Tea
package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
)

// AddProfileModel represents the add new profile form UI
type AddProfileModel struct {
	inputs          []textinput.Model
	focusedField    int
	width           int
	height          int
	showPreview     bool
	createdProfile  *fingerprints.Profile
	profileName     string
}

// Input field indices
const (
	FieldProfileName = iota
	FieldJA4R
	FieldUserAgent
	FieldDescription
	FieldPlatform
	FieldTLSVersion
	FieldHTTPVersion
	FieldCount // Total number of fields
)

// NewAddProfileModel creates a new add profile form model
func NewAddProfileModel() AddProfileModel {
	inputs := make([]textinput.Model, FieldCount)
	
	// Profile Name
	inputs[FieldProfileName] = textinput.New()
	inputs[FieldProfileName].Placeholder = "e.g., custom_chrome140"
	inputs[FieldProfileName].Focus()
	inputs[FieldProfileName].CharLimit = 50
	inputs[FieldProfileName].Width = 50
	
	// JA4R
	inputs[FieldJA4R] = textinput.New()
	inputs[FieldJA4R].Placeholder = "e.g., t13d1516h2_1301,1302,1303..."
	inputs[FieldJA4R].CharLimit = 500
	inputs[FieldJA4R].Width = 80
	
	// User Agent
	inputs[FieldUserAgent] = textinput.New()
	inputs[FieldUserAgent].Placeholder = "e.g., Mozilla/5.0 (X11; Linux x86_64)..."
	inputs[FieldUserAgent].CharLimit = 300
	inputs[FieldUserAgent].Width = 80
	
	// Description
	inputs[FieldDescription] = textinput.New()
	inputs[FieldDescription].Placeholder = "e.g., Custom Chrome 140 on Linux"
	inputs[FieldDescription].CharLimit = 100
	inputs[FieldDescription].Width = 60
	
	// Platform
	inputs[FieldPlatform] = textinput.New()
	inputs[FieldPlatform].Placeholder = "e.g., Linux, Windows, macOS, iOS, Android"
	inputs[FieldPlatform].CharLimit = 20
	inputs[FieldPlatform].Width = 30
	
	// TLS Version
	inputs[FieldTLSVersion] = textinput.New()
	inputs[FieldTLSVersion].Placeholder = "e.g., 1.3"
	inputs[FieldTLSVersion].SetValue("1.3")
	inputs[FieldTLSVersion].CharLimit = 10
	inputs[FieldTLSVersion].Width = 15
	
	// HTTP Version
	inputs[FieldHTTPVersion] = textinput.New()
	inputs[FieldHTTPVersion].Placeholder = "e.g., h2"
	inputs[FieldHTTPVersion].SetValue("h2")
	inputs[FieldHTTPVersion].CharLimit = 10
	inputs[FieldHTTPVersion].Width = 15
	
	return AddProfileModel{
		inputs:       inputs,
		focusedField: 0,
	}
}

// Init initializes the add profile model
func (m AddProfileModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the add profile model
func (m AddProfileModel) Update(msg tea.Msg) (AddProfileModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focusedField = (m.focusedField + 1) % len(m.inputs)
			m.updateFocus()
			return m, nil
			
		case "shift+tab":
			m.focusedField = (m.focusedField - 1 + len(m.inputs)) % len(m.inputs)
			m.updateFocus()
			return m, nil
			
		case "enter":
			if m.showPreview {
				// In preview mode, Enter goes back to edit mode
				m.showPreview = false
				return m, nil
			} else if m.validateInputs() {
				// In edit mode, Enter creates profile and shows preview
				m.createProfile()
				m.showPreview = true
				return m, nil
			}
			// If validation fails, don't consume the enter key
			// Let it fall through to update the focused input
			
		case "ctrl+s":
			// TODO: Save profile (would need to be integrated with profile storage)
			return m, nil
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	m.inputs[m.focusedField], cmd = m.inputs[m.focusedField].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateFocus updates which input field has focus
func (m *AddProfileModel) updateFocus() {
	for i, input := range m.inputs {
		if i == m.focusedField {
			input.Focus()
		} else {
			input.Blur()
		}
		m.inputs[i] = input
	}
}

// validateInputs checks if all required fields are filled
func (m *AddProfileModel) validateInputs() bool {
	// Profile name is required
	if strings.TrimSpace(m.inputs[FieldProfileName].Value()) == "" {
		return false
	}
	
	// JA4R is required
	if strings.TrimSpace(m.inputs[FieldJA4R].Value()) == "" {
		return false
	}
	
	// User Agent is required
	if strings.TrimSpace(m.inputs[FieldUserAgent].Value()) == "" {
		return false
	}
	
	return true
}

// createProfile creates a new profile from the form inputs
func (m *AddProfileModel) createProfile() {
	m.profileName = strings.TrimSpace(m.inputs[FieldProfileName].Value())
	
	profile := &fingerprints.Profile{
		JA3:         "", // JA3 is deprecated, leave empty
		JA4R:        strings.TrimSpace(m.inputs[FieldJA4R].Value()),
		UserAgent:   strings.TrimSpace(m.inputs[FieldUserAgent].Value()),
		Description: strings.TrimSpace(m.inputs[FieldDescription].Value()),
		Platform:    strings.TrimSpace(m.inputs[FieldPlatform].Value()),
		TLSVersion:  strings.TrimSpace(m.inputs[FieldTLSVersion].Value()),
		HTTPVersion: strings.TrimSpace(m.inputs[FieldHTTPVersion].Value()),
	}
	
	// Set defaults if empty
	if profile.Description == "" {
		profile.Description = fmt.Sprintf("Custom profile: %s", m.profileName)
	}
	if profile.Platform == "" {
		profile.Platform = "Custom"
	}
	if profile.TLSVersion == "" {
		profile.TLSVersion = "1.3"
	}
	if profile.HTTPVersion == "" {
		profile.HTTPVersion = "h2"
	}
	
	m.createdProfile = profile
}

// View renders the add profile form
func (m AddProfileModel) View() string {
	if m.showPreview {
		return m.renderPreview()
	}

	var content strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98D8C8")).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("➕ Add New Browser Profile")
	content.WriteString(header)
	content.WriteString("\n\n")

	// Form fields
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true)

	requiredStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E06C75"))

	focusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true)

	fieldLabels := []string{
		"Profile Name",
		"JA4R Fingerprint", 
		"User Agent",
		"Description",
		"Platform",
		"TLS Version",
		"HTTP Version",
	}

	requiredFields := []bool{true, true, true, false, false, false, false}

	for i, label := range fieldLabels {
		// Field label
		if i == m.focusedField {
			content.WriteString(focusedStyle.Render(fmt.Sprintf("► %s", label)))
		} else {
			content.WriteString(labelStyle.Render(fmt.Sprintf("  %s", label)))
		}
		
		if requiredFields[i] {
			content.WriteString(requiredStyle.Render(" *"))
		}
		content.WriteString("\n")

		// Input field
		content.WriteString("  ")
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n\n")
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98D8C8")).
		Background(lipgloss.Color("#2D3748")).
		Padding(1).
		MarginTop(1)

	instructions := instructionStyle.Render(`Tips:
• Get JA4R fingerprints from browser developer tools or fingerprinting sites
• User Agent should match the browser/platform you're emulating
• Platform examples: Linux, Windows, macOS, iOS, Android`)

	content.WriteString(instructions)
	content.WriteString("\n\n")

	// Validation status
	if !m.validateInputs() {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E06C75"))
		
		content.WriteString(errorStyle.Render("⚠ Please fill in all required fields (*) before creating the profile"))
		content.WriteString("\n\n")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F"))

	var footerText string
	if m.validateInputs() {
		footerText = "TAB/Shift+TAB to navigate • ENTER to create profile • ESC to return"
	} else {
		footerText = "TAB/Shift+TAB to navigate • Fill required fields (*) then ENTER • ESC to return"
	}
	footer := footerStyle.Render(footerText)
	content.WriteString(footer)

	return content.String()
}

// renderPreview renders the profile preview
func (m AddProfileModel) renderPreview() string {
	var content strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98D8C8")).
		Bold(true).
		Padding(0, 1)

	content.WriteString(headerStyle.Render("✅ Profile Created Successfully"))
	content.WriteString("\n\n")

	if m.createdProfile != nil {
		// Profile summary
		summaryStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6C7B7F")).
			Padding(1).
			MarginBottom(1)

		summary := fmt.Sprintf(`Profile Name: %s
Description: %s
Platform: %s
TLS Version: %s
HTTP Version: %s

JA4R: %s

User Agent: %s`, 
			m.profileName,
			m.createdProfile.Description,
			m.createdProfile.Platform,
			m.createdProfile.TLSVersion,
			m.createdProfile.HTTPVersion,
			m.createdProfile.JA4R,
			m.createdProfile.UserAgent)

		content.WriteString(summaryStyle.Render(summary))
		content.WriteString("\n")
	}

	// Instructions
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB"))

	content.WriteString(helpStyle.Render("Note: This profile has been created in memory for this session."))
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("To persist profiles permanently, integration with profile storage is needed."))
	content.WriteString("\n\n")

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F"))

	footer := footerStyle.Render("ENTER to edit profile • ESC to return to profile browser")
	content.WriteString(footer)

	return content.String()
}