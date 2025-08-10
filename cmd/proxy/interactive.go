// interactive.go - Interactive Bubble Tea application for CycleTLS-Proxy
package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/cmd/proxy/models"
	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
)

// AppMode represents the different modes of the application
type AppMode int

const (
	ModeMenu AppMode = iota
	ModeHelp
	ModeProfiles
	ModeServer
	ModeConfig
	ModeMonitor
	ModeTest
)

// InteractiveApp is the main Bubble Tea model for the interactive application
type InteractiveApp struct {
	mode        AppMode
	width       int
	height      int
	logger      *log.Logger
	port        string
	profiles    map[string]fingerprints.Profile
	
	// Sub-models
	helpModel    models.HelpModel
	profileModel models.ProfileModel
	configModel  models.ConfigModel
	monitorModel models.MonitorModel
	
	// State
	quitting     bool
	startServer  bool
	serverMode   bool
}

// NewInteractiveApp creates a new interactive application
func NewInteractiveApp(port string, logger *log.Logger) *InteractiveApp {
	profiles := fingerprints.GetDefaultProfiles()
	
	return &InteractiveApp{
		mode:     ModeMenu,
		logger:   logger,
		port:     port,
		profiles: profiles,
		
		helpModel:    models.NewHelpModel(),
		profileModel: models.NewProfileModel(profiles),
		configModel:  models.NewConfigModel(),
		monitorModel: models.NewMonitorModel(),
	}
}

// Init initializes the Bubble Tea application
func (m *InteractiveApp) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.EnableMouseAllMotion,
	)
}

// Update handles messages and updates the model state
func (m *InteractiveApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
		
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}
	
	// Delegate to sub-models based on current mode
	switch m.mode {
	case ModeHelp:
		var cmd tea.Cmd
		m.helpModel, cmd = m.helpModel.Update(msg)
		return m, cmd
		
	case ModeProfiles:
		var cmd tea.Cmd
		m.profileModel, cmd = m.profileModel.Update(msg)
		return m, cmd
		
	case ModeConfig:
		var cmd tea.Cmd
		m.configModel, cmd = m.configModel.Update(msg)
		return m, cmd
		
	case ModeMonitor:
		var cmd tea.Cmd
		m.monitorModel, cmd = m.monitorModel.Update(msg)
		return m, cmd
	}
	
	return m, nil
}

// handleKeyPress processes keyboard input
func (m *InteractiveApp) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	
	// Global key handlers
	switch key {
	case "q", "ctrl+c":
		if m.mode == ModeMenu {
			m.quitting = true
			return m, tea.Quit
		} else {
			m.mode = ModeMenu
			return m, nil
		}
		
	case "esc":
		if m.mode != ModeMenu {
			m.mode = ModeMenu
			return m, nil
		}
	}
	
	// Mode-specific key handlers
	switch m.mode {
	case ModeMenu:
		return m.handleMenuKeys(key)
	default:
		// Let sub-models handle their own keys
		return m, nil
	}
}

// handleMenuKeys processes menu-specific keyboard input
func (m *InteractiveApp) handleMenuKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "h", "?":
		m.mode = ModeHelp
		return m, nil
		
	case "p":
		m.mode = ModeProfiles
		return m, nil
		
	case "s", "enter":
		m.startServer = true
		m.serverMode = true
		return m, tea.Quit
		
	case "c":
		m.mode = ModeConfig
		return m, nil
		
	case "m":
		m.mode = ModeMonitor
		return m, nil
		
	case "t":
		m.mode = ModeTest
		return m, nil
	}
	
	return m, nil
}

// View renders the current view
func (m *InteractiveApp) View() string {
	if m.quitting {
		return m.renderGoodbye()
	}
	
	switch m.mode {
	case ModeMenu:
		return m.renderMenu()
	case ModeHelp:
		return m.renderHelp()
	case ModeProfiles:
		return m.renderProfiles()
	case ModeConfig:
		return m.renderConfig()
	case ModeMonitor:
		return m.renderMonitor()
	case ModeTest:
		return m.renderTest()
	default:
		return m.renderMenu()
	}
}

// renderMenu renders the main menu
func (m *InteractiveApp) renderMenu() string {
	var b strings.Builder
	
	// Title banner
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(1)
	
	title := titleStyle.Render(m.getASCIIBanner())
	
	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(2)
	
	subtitle := subtitleStyle.Render("Advanced TLS Fingerprint Proxy Server")
	
	// Example command
	exampleStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		MarginBottom(2).
		Width(m.width - 4).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)
	
	// Create a gradient-like effect for the example box
	exampleHeaderStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Bold(true)
	
	exampleCommandStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 1).
		MarginTop(1)
	
	exampleContent := fmt.Sprintf(
		"%s\n%s",
		exampleHeaderStyle.Render("🚀 Quick Start Example:"),
		exampleCommandStyle.Render(
			fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-IDENTIFIER: chrome" http://localhost:%s`, m.port),
		),
	)
	
	example := exampleStyle.Render(exampleContent)
	
	// Menu options
	menuStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(successColor).
		Padding(1, 2).
		Width(m.width - 4)
	
	keyStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Background(lipgloss.Color("#2a2a2a")).
		Padding(0, 1).
		MarginRight(1)
	
	inactiveKeyStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 1).
		MarginRight(1)
	
	menuItems := []string{
		fmt.Sprintf("%s %s Start the proxy server", 
			keyStyle.Render(" s "), "🚀"),
		fmt.Sprintf("%s %s View interactive help & documentation", 
			keyStyle.Render(" h "), "📚"),
		fmt.Sprintf("%s %s Browse browser profiles (%d available)", 
			keyStyle.Render(" p "), "🌐", len(m.profiles)),
		fmt.Sprintf("%s %s Configure settings", 
			keyStyle.Render(" c "), "⚙️"),
		fmt.Sprintf("%s %s Test requests", 
			keyStyle.Render(" t "), "🧪"),
		fmt.Sprintf("%s %s Monitor (when server is running)", 
			inactiveKeyStyle.Render(" m "), "📊"),
		fmt.Sprintf("%s %s Quit application", 
			keyStyle.Render(" q "), "👋"),
	}
	
	menu := menuStyle.Render(strings.Join(menuItems, "\n"))
	
	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Align(lipgloss.Center).
		Width(m.width).
		MarginTop(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("#333333")).
		PaddingTop(1)
	
	footer := footerStyle.Render(fmt.Sprintf("🌍 Listening on localhost:%s • ⚡ Ready to serve TLS fingerprinted requests • 🔒 %d profiles available", m.port, len(m.profiles)))
	
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(subtitle)
	b.WriteString("\n")
	b.WriteString(example)
	b.WriteString("\n")
	b.WriteString(menu)
	b.WriteString("\n")
	b.WriteString(footer)
	
	return b.String()
}

// renderHelp renders the help view
func (m *InteractiveApp) renderHelp() string {
	header := m.renderModeHeader("Interactive Help & Documentation", "Press [esc] to return to menu")
	content := m.helpModel.View()
	return header + "\n" + content
}

// renderProfiles renders the profiles view
func (m *InteractiveApp) renderProfiles() string {
	header := m.renderModeHeader("Browser Profiles", "Press [esc] to return to menu")
	content := m.profileModel.View()
	return header + "\n" + content
}

// renderConfig renders the configuration view
func (m *InteractiveApp) renderConfig() string {
	header := m.renderModeHeader("Configuration", "Press [esc] to return to menu")
	content := m.configModel.View()
	return header + "\n" + content
}

// renderMonitor renders the monitoring view
func (m *InteractiveApp) renderMonitor() string {
	header := m.renderModeHeader("Live Monitor", "Press [esc] to return to menu")
	content := m.monitorModel.View()
	return header + "\n" + content
}

// renderTest renders the test view
func (m *InteractiveApp) renderTest() string {
	header := m.renderModeHeader("Request Testing", "Press [esc] to return to menu")
	content := "Test mode - Coming soon!\n\nThis will allow you to:\n• Send test requests\n• Verify TLS fingerprints\n• Test different browser profiles\n• Validate proxy functionality"
	return header + "\n" + content
}

// renderModeHeader renders a consistent header for different modes
func (m *InteractiveApp) renderModeHeader(title, subtitle string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Align(lipgloss.Center).
		Width(m.width)
	
	subtitleStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(1)
	
	return titleStyle.Render(title) + "\n" + subtitleStyle.Render(subtitle)
}

// renderGoodbye renders the goodbye message
func (m *InteractiveApp) renderGoodbye() string {
	style := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.width).
		MarginTop(2)
	
	return style.Render("Thank you for using CycleTLS-Proxy! 👋")
}

// getASCIIBanner returns the ASCII art banner
func (m *InteractiveApp) getASCIIBanner() string {
	return `
 ██████╗██╗   ██╗ ██████╗██╗     ███████╗████████╗██╗     ███████╗
██╔════╝╚██╗ ██╔╝██╔════╝██║     ██╔════╝╚══██╔══╝██║     ██╔════╝
██║      ╚████╔╝ ██║     ██║     █████╗     ██║   ██║     ███████╗
██║       ╚██╔╝  ██║     ██║     ██╔══╝     ██║   ██║     ╚════██║
╚██████╗   ██║   ╚██████╗███████╗███████╗   ██║   ███████╗███████║
 ╚═════╝   ╚═╝    ╚═════╝╚══════╝╚══════╝   ╚═╝   ╚══════╝╚══════╝

             ██████╗ ██████╗  ██████╗ ██╗  ██╗██╗   ██╗
             ██╔══██╗██╔══██╗██╔═══██╗╚██╗██╔╝╚██╗ ██╔╝
             ██████╔╝██████╔╝██║   ██║ ╚███╔╝  ╚████╔╝ 
             ██╔═══╝ ██╔══██╗██║   ██║ ██╔██╗   ╚██╔╝  
             ██║     ██║  ██║╚██████╔╝██╔╝ ██╗   ██║   
             ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   `
}

// ShouldStartServer returns true if the user chose to start the server
func (m *InteractiveApp) ShouldStartServer() bool {
	return m.startServer
}

// IsServerMode returns true if the app should run in server mode
func (m *InteractiveApp) IsServerMode() bool {
	return m.serverMode
}