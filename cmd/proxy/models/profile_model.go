// profile_model.go - Browser profile viewer model for Bubble Tea
package models

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
)

// ProfileModel represents the browser profile viewer
type ProfileModel struct {
	list     list.Model
	profiles map[string]fingerprints.Profile
	selected string
	width    int
	height   int
}

// ProfileItem represents a profile item in the list
type ProfileItem struct {
	name    string
	profile fingerprints.Profile
}

// FilterValue returns the filter value for the item
func (i ProfileItem) FilterValue() string { 
	return i.name + " " + i.profile.Description + " " + i.profile.Platform
}

// Title returns the title for display
func (i ProfileItem) Title() string { 
	return fmt.Sprintf("%s", i.name)
}

// Description returns the description for display
func (i ProfileItem) Description() string { 
	return fmt.Sprintf("%s • %s • %s", 
		i.profile.Description, 
		i.profile.Platform, 
		i.profile.TLSVersion)
}

// NewProfileModel creates a new profile browser model
func NewProfileModel(profiles map[string]fingerprints.Profile) ProfileModel {
	// Convert profiles to list items
	items := make([]list.Item, 0, len(profiles))
	
	// Sort profile names for consistent ordering
	var names []string
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	
	for _, name := range names {
		items = append(items, ProfileItem{
			name:    name,
			profile: profiles[name],
		})
	}
	
	// Create list with custom styling
	l := list.New(items, newProfileDelegate(), 0, 0)
	l.Title = "Browser Profiles"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true).
		Padding(0, 1)
	
	return ProfileModel{
		list:     l,
		profiles: profiles,
	}
}

// Init initializes the profile model
func (m ProfileModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the profile model
func (m ProfileModel) Update(msg tea.Msg) (ProfileModel, tea.Cmd) {
	// Handle special cases first following Charmbracelet pattern
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Set list dimensions with proper padding
		m.list.SetSize(msg.Width-4, msg.Height-6)
		
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(ProfileItem); ok {
				m.selected = item.name
			}
			// Don't return here - let the list handle the enter key too
		}
	}
	
	// Always forward ALL messages to the list for proper navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	
	return m, cmd
}

// View renders the profile model
func (m ProfileModel) View() string {
	var content strings.Builder
	
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true).
		Padding(0, 1)
	
	header := headerStyle.Render(fmt.Sprintf("🌐 %d Browser Profiles Available", len(m.profiles)))
	content.WriteString(header)
	content.WriteString("\n\n")
	
	// Split view: list on left, details on right
	if m.width > 80 {
		listWidth := m.width / 2
		detailWidth := m.width - listWidth - 2
		
		m.list.SetWidth(listWidth)
		
		listView := m.list.View()
		detailView := m.renderProfileDetail(detailWidth)
		
		// Side-by-side layout
		listStyle := lipgloss.NewStyle().Width(listWidth)
		detailStyle := lipgloss.NewStyle().
			Width(detailWidth).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#6C7B7F")).
			PaddingLeft(1)
		
		content.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				listStyle.Render(listView),
				detailStyle.Render(detailView),
			),
		)
	} else {
		// Stacked layout for narrow screens
		content.WriteString(m.list.View())
		
		if m.selected != "" {
			content.WriteString("\n")
			content.WriteString(m.renderProfileDetail(m.width))
		}
	}
	
	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F")).
		Padding(1, 1, 0, 1)
	
	footer := footerStyle.Render("↑/↓ to navigate • ENTER to select • / to filter • ESC to return")
	content.WriteString("\n")
	content.WriteString(footer)
	
	return content.String()
}

// renderProfileDetail renders detailed information about a profile
func (m ProfileModel) renderProfileDetail(width int) string {
	var profileName string
	
	// Get currently selected profile
	if item, ok := m.list.SelectedItem().(ProfileItem); ok {
		profileName = item.name
	} else if m.selected != "" {
		profileName = m.selected
	} else {
		return lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color("#6C7B7F")).
			Render("Select a profile to view details")
	}
	
	profile, exists := m.profiles[profileName]
	if !exists {
		return "Profile not found"
	}
	
	var content strings.Builder
	
	// Profile title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true).
		MarginBottom(1)
	
	content.WriteString(titleStyle.Render(fmt.Sprintf("📱 %s", profileName)))
	content.WriteString("\n\n")
	
	// Profile details
	detailStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))
	
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true)
	
	details := []struct {
		label string
		value string
	}{
		{"Description", profile.Description},
		{"Platform", profile.Platform},
		{"HTTP Version", profile.HTTPVersion},
		{"TLS Version", profile.TLSVersion},
		{"User Agent", profile.UserAgent},
		{"JA3 Fingerprint", profile.JA3},
	}
	
	for _, detail := range details {
		content.WriteString(labelStyle.Render(detail.label + ":"))
		content.WriteString("\n")
		
		// Wrap long values
		value := detail.value
		if len(value) > width-4 && detail.label != "User Agent" && detail.label != "JA3 Fingerprint" {
			// Don't wrap these as they need to stay intact
		} else if len(value) > width-4 {
			// Simple word wrap for long values
			value = wordWrap(value, width-4)
		}
		
		content.WriteString(detailStyle.Render(value))
		content.WriteString("\n\n")
	}
	
	// Example usage
	exampleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98D8C8")).
		Background(lipgloss.Color("#2D3748")).
		Padding(1).
		MarginTop(1)
	
	example := fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" \
     -H "X-IDENTIFIER: %s" \
     http://localhost:8080`, profileName)
	
	content.WriteString(labelStyle.Render("Example Usage:"))
	content.WriteString("\n")
	content.WriteString(exampleStyle.Render(example))
	
	return content.String()
}

// newProfileDelegate creates a custom delegate for the profile list
func newProfileDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true)
	
	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C678DD"))
	
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))
	
	d.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F"))
	
	d.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F"))
	
	d.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4A5568"))
	
	return d
}

// wordWrap wraps text to the specified width
func wordWrap(text string, width int) string {
	if len(text) <= width {
		return text
	}
	
	var result strings.Builder
	words := strings.Fields(text)
	line := ""
	
	for _, word := range words {
		if len(line)+len(word)+1 > width {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
			line = word
		} else {
			if line != "" {
				line += " "
			}
			line += word
		}
	}
	
	if line != "" {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(line)
	}
	
	return result.String()
}