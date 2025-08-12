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

// SwitchToAddProfileMsg is a message to request switching to add profile mode
type SwitchToAddProfileMsg struct{}

// ProfileModel represents the browser profile viewer with rotation selection
type ProfileModel struct {
	list              list.Model
	profiles          map[string]fingerprints.Profile
	selectedProfiles  map[string]bool  // tracks which profiles are selected for rotation
	rotationEnabled   bool             // whether rotation is enabled
	sessionSticky     bool             // whether sessions stick to same profile
	width             int
	height            int
	rotator           *fingerprints.ProfileRotator // reference to the rotator
}

// ProfileItem represents a profile item in the list
type ProfileItem struct {
	name       string
	profile    fingerprints.Profile
	selected   bool   // whether this profile is selected for rotation
	isAddNew   bool   // special flag for "Add New Profile" item
}

// FilterValue returns the filter value for the item
func (i ProfileItem) FilterValue() string {
	if i.isAddNew {
		return "add new profile"
	}
	return i.name + " " + i.profile.Description + " " + i.profile.Platform
}

// Title returns the title for display
func (i ProfileItem) Title() string {
	if i.isAddNew {
		return "➕ Add New Profile"
	}
	
	checkbox := "❌"
	if i.selected {
		checkbox = "✅"
	}
	return fmt.Sprintf("%s %s", checkbox, i.name)
}

// Description returns the description for display
func (i ProfileItem) Description() string {
	if i.isAddNew {
		return "Create a custom browser profile"
	}
	
	return fmt.Sprintf("%s • %s • %s",
		i.profile.Description,
		i.profile.Platform,
		i.profile.TLSVersion)
}

// NewProfileModel creates a new profile browser model with rotation support
func NewProfileModel(profiles map[string]fingerprints.Profile, rotator *fingerprints.ProfileRotator) ProfileModel {
	// Initialize selection state
	selectedProfiles := make(map[string]bool)
	
	// Default selection: chrome138 and chrome139
	selectedProfiles["chrome138"] = true
	selectedProfiles["chrome139"] = true
	
	// Get current rotator config if available
	rotationEnabled := true
	sessionSticky := true
	if rotator != nil {
		config := rotator.GetConfig()
		rotationEnabled = config.RotationEnabled
		sessionSticky = config.SessionSticky
		
		// Update selection from rotator config
		selectedProfiles = make(map[string]bool)
		for _, profileName := range config.EnabledProfiles {
			selectedProfiles[profileName] = true
		}
	}
	
	// Convert profiles to list items
	items := make([]list.Item, 0, len(profiles)+1) // +1 for "Add New" option
	
	// Sort profile names for consistent ordering
	var names []string
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	
	// Add regular profiles with selection state
	for _, name := range names {
		items = append(items, ProfileItem{
			name:     name,
			profile:  profiles[name],
			selected: selectedProfiles[name],
			isAddNew: false,
		})
	}
	
	// Add "Add New Profile" option at the end
	items = append(items, ProfileItem{
		name:     "add_new",
		profile:  fingerprints.Profile{},
		selected: false,
		isAddNew: true,
	})

	// Create list with custom styling
	l := list.New(items, newProfileDelegate(), 0, 0)
	l.Title = "Browser Profiles & Rotation"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true).
		Padding(0, 1)

	return ProfileModel{
		list:             l,
		profiles:         profiles,
		selectedProfiles: selectedProfiles,
		rotationEnabled:  rotationEnabled,
		sessionSticky:    sessionSticky,
		rotator:          rotator,
	}
}

// Init initializes the profile model
func (m ProfileModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the profile model
func (m ProfileModel) Update(msg tea.Msg) (ProfileModel, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Forward mouse events to list for click/hover functionality
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Set list dimensions with proper padding for header
		m.list.SetSize(msg.Width-4, msg.Height-8)

	case tea.KeyMsg:
		switch msg.String() {
		case " ", "space":
			// Toggle profile selection for rotation
			if item, ok := m.list.SelectedItem().(ProfileItem); ok {
				if item.isAddNew {
					// Switch to add profile mode
					return m, func() tea.Msg { return SwitchToAddProfileMsg{} }
				}
				
				// Toggle selection
				m.selectedProfiles[item.name] = !m.selectedProfiles[item.name]
				m.updateListItems()
				m.syncWithRotator()
			}
			return m, tea.Batch(cmds...)
			
		case "r":
			// Toggle rotation enabled/disabled
			m.rotationEnabled = !m.rotationEnabled
			m.syncWithRotator()
			return m, tea.Batch(cmds...)
			
		case "s":
			// Toggle session sticky
			m.sessionSticky = !m.sessionSticky
			m.syncWithRotator()
			return m, tea.Batch(cmds...)
			
		case "enter":
			if item, ok := m.list.SelectedItem().(ProfileItem); ok {
				if item.isAddNew {
					// Switch to add profile mode
					return m, func() tea.Msg { return SwitchToAddProfileMsg{} }
				}
				// For regular profiles, enter toggles selection (same as space)
				m.selectedProfiles[item.name] = !m.selectedProfiles[item.name]
				m.updateListItems()
				m.syncWithRotator()
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Forward other messages to the list for navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateListItems updates the list items with current selection state
func (m *ProfileModel) updateListItems() {
	items := make([]list.Item, 0, len(m.profiles)+1)
	
	// Sort profile names for consistent ordering
	var names []string
	for name := range m.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	
	// Add regular profiles with updated selection state
	for _, name := range names {
		items = append(items, ProfileItem{
			name:     name,
			profile:  m.profiles[name],
			selected: m.selectedProfiles[name],
			isAddNew: false,
		})
	}
	
	// Add "Add New Profile" option at the end
	items = append(items, ProfileItem{
		name:     "add_new",
		profile:  fingerprints.Profile{},
		selected: false,
		isAddNew: true,
	})
	
	m.list.SetItems(items)
}

// syncWithRotator syncs the current state with the rotator
func (m *ProfileModel) syncWithRotator() {
	if m.rotator == nil {
		return
	}
	
	// Get list of selected profiles
	var enabledProfiles []string
	for profileName, selected := range m.selectedProfiles {
		if selected {
			enabledProfiles = append(enabledProfiles, profileName)
		}
	}
	
	// Update rotator config
	config := &fingerprints.RotationConfig{
		EnabledProfiles: enabledProfiles,
		RotationEnabled: m.rotationEnabled,
		SessionSticky:   m.sessionSticky,
	}
	
	m.rotator.UpdateConfig(config)
}

// View renders the profile model
func (m ProfileModel) View() string {
	var content strings.Builder

	// Header with rotation status
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true).
		Padding(0, 1)

	// Count selected profiles
	selectedCount := 0
	for _, selected := range m.selectedProfiles {
		if selected {
			selectedCount++
		}
	}
	
	rotationStatus := "OFF"
	if m.rotationEnabled {
		rotationStatus = "ON"
	}
	
	stickyStatus := ""
	if m.sessionSticky {
		stickyStatus = " • Session Sticky"
	}
	
	header := headerStyle.Render(fmt.Sprintf("🎲 Random Rotation: %s | %d profiles selected%s", 
		rotationStatus, selectedCount, stickyStatus))
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

		// Always show detail for narrow screens
		content.WriteString("\n")
		content.WriteString(m.renderProfileDetail(m.width))
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F")).
		Padding(1, 1, 0, 1)

	footer := footerStyle.Render("↑/↓ navigate • SPACE/ENTER toggle selection • R toggle rotation • S toggle sticky • / filter • ESC return")
	content.WriteString("\n")
	content.WriteString(footer)

	return content.String()
}

// renderProfileDetail renders detailed information about a profile
func (m ProfileModel) renderProfileDetail(width int) string {
	// Get currently selected item
	item, ok := m.list.SelectedItem().(ProfileItem)
	if !ok {
		return lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color("#6C7B7F")).
			Render("Select a profile to view details")
	}

	// Handle "Add New Profile" case
	if item.isAddNew {
		var content strings.Builder
		
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B9D")).
			Bold(true).
			MarginBottom(1)
		
		content.WriteString(titleStyle.Render("➕ Add New Profile"))
		content.WriteString("\n\n")
		
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98D8C8"))
		
		help := helpStyle.Render(`Create a custom browser profile with your own:
• JA4R fingerprint
• User Agent string  
• TLS version
• Platform information

Press ENTER to open the profile creator.`)
		
		content.WriteString(help)
		return content.String()
	}

	profile, exists := m.profiles[item.name]
	if !exists {
		return "Profile not found"
	}

	var content strings.Builder

	// Profile title with selection status
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true).
		MarginBottom(1)

	selectedIcon := "❌"
	if item.selected {
		selectedIcon = "✅"
	}

	content.WriteString(titleStyle.Render(fmt.Sprintf("%s 📱 %s", selectedIcon, item.name)))
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
		{"JA4R Fingerprint", profile.JA4R},
		{"JA3 Fingerprint", profile.JA3},
	}

	for _, detail := range details {
		content.WriteString(labelStyle.Render(detail.label + ":"))
		content.WriteString("\n")

		// Wrap long values
		value := detail.value
		if len(value) > width-4 && detail.label != "User Agent" && detail.label != "JA3 Fingerprint" && detail.label != "JA4R Fingerprint" {
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
     http://localhost:8080`, item.name)

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
