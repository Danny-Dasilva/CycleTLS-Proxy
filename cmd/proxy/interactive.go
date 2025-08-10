// interactive.go - Interactive Bubble Tea application for CycleTLS-Proxy
package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/cmd/proxy/models"
	"github.com/Danny-Dasilva/CycleTLS-Proxy/cmd/proxy/styles"
	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/proxy"
)

// AppMode represents the different modes of the application
type AppMode int

const (
	ModeMenu AppMode = iota
	ModeParamBrowser
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
	paramBrowserModel models.ParamBrowserModel
	profileModel      models.ProfileModel
	configModel       models.ConfigModel
	monitorModel      models.MonitorModel
	logo              *models.GradientLogo
	
	// State
	quitting     bool
	startServer  bool
	serverMode   bool
}

// NewInteractiveApp creates a new interactive application
func NewInteractiveApp(port string, logger *log.Logger, handler ...*proxy.Handler) *InteractiveApp {
	profiles := fingerprints.GetDefaultProfiles()
	
	app := &InteractiveApp{
		mode:     ModeMenu,
		logger:   logger,
		port:     port,
		profiles: profiles,
		
		paramBrowserModel: models.NewParamBrowserModel(port),
		profileModel:      models.NewProfileModel(profiles),
		configModel:       models.NewConfigModel(),
		monitorModel:      models.NewMonitorModel(),
		logo:              models.NewGradientLogo(0, 0),
	}
	
	// If handler is provided, we're in server mode
	if len(handler) > 0 && handler[0] != nil {
		app.serverMode = true
	}
	
	return app
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
		
		// Update logo dimensions
		if m.logo != nil {
			m.logo = models.NewGradientLogo(msg.Width, msg.Height)
		}
		
		return m, nil
		
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}
	
	// Delegate to sub-models based on current mode
	switch m.mode {
	case ModeParamBrowser:
		var cmd tea.Cmd
		m.paramBrowserModel, cmd = m.paramBrowserModel.Update(msg)
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
		m.mode = ModeParamBrowser
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
	case ModeParamBrowser:
		return m.renderParamBrowser()
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
	
	// Render gradient logo
	var title string
	if m.width < 80 || m.height < 25 {
		// Use compact version for small terminals
		title = m.logo.RenderCompact()
	} else {
		// Use full gradient logo
		title = m.logo.RenderWithSubtitle("Advanced TLS Fingerprint Proxy Server")
	}
	
	// Server status
	var statusLine string
	if m.serverMode {
		statusStyle := styles.SuccessStyle.Copy().
			Align(lipgloss.Center).
			Width(m.width).
			MarginBottom(1)
		statusLine = statusStyle.Render("🟢 SERVER RUNNING • Ready to accept requests")
	}
	
	// Example command with responsive width
	exampleWidth := m.width - 4
	if exampleWidth < 60 {
		exampleWidth = 60
	}
	
	exampleStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentOrange).
		Padding(1, 2).
		MarginBottom(2).
		Width(exampleWidth).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)
	
	exampleHeaderStyle := styles.WarningStyle.Copy()
	exampleCommandStyle := styles.CodeStyle.Copy().MarginTop(1)
	
	var exampleCmd string
	if m.width < 100 {
		// Short version for small terminals
		exampleCmd = fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" \
     http://localhost:%s`, m.port)
	} else {
		// Full version
		exampleCmd = fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-IDENTIFIER: chrome" http://localhost:%s`, m.port)
	}
	
	exampleContent := fmt.Sprintf(
		"%s\n%s",
		exampleHeaderStyle.Render("🚀 Quick Start Example:"),
		exampleCommandStyle.Render(exampleCmd),
	)
	
	example := exampleStyle.Render(exampleContent)
	
	// Menu options with responsive sizing
	menuWidth := m.width - 4
	if menuWidth < 60 {
		menuWidth = 60
	}
	
	menuStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentGreen).
		Padding(1, 2).
		Width(menuWidth)
	
	var menuItems []string
	if m.serverMode {
		menuItems = []string{
			fmt.Sprintf("%s %s Browse all parameters & examples", 
				styles.KeyStyle("h"), "📋"),
			fmt.Sprintf("%s %s Browse browser profiles (%d available)", 
				styles.KeyStyle("p"), "🌐", len(m.profiles)),
			fmt.Sprintf("%s %s Configure settings", 
				styles.KeyStyle("c"), "⚙️"),
			fmt.Sprintf("%s %s Test requests with running server", 
				styles.KeyStyle("t"), "🧪"),
			fmt.Sprintf("%s %s Live monitoring dashboard", 
				styles.KeyStyle("m"), "📊"),
			fmt.Sprintf("%s %s Quit application (stops server)", 
				styles.KeyStyle("q"), "👋"),
		}
	} else {
		inactiveKeyStyle := lipgloss.NewStyle().
			Foreground(styles.TextDisabled).
			Background(styles.BgTertiary).
			Padding(0, 1).
			MarginRight(1)
		
		menuItems = []string{
			fmt.Sprintf("%s %s Start the proxy server", 
				styles.KeyStyle("s"), "🚀"),
			fmt.Sprintf("%s %s Browse all parameters & examples", 
				styles.KeyStyle("h"), "📋"),
			fmt.Sprintf("%s %s Browse browser profiles (%d available)", 
				styles.KeyStyle("p"), "🌐", len(m.profiles)),
			fmt.Sprintf("%s %s Configure settings", 
				styles.KeyStyle("c"), "⚙️"),
			fmt.Sprintf("%s %s Test requests", 
				inactiveKeyStyle.Render(" t "), "🧪"),
			fmt.Sprintf("%s %s Monitor (requires running server)", 
				inactiveKeyStyle.Render(" m "), "📊"),
			fmt.Sprintf("%s %s Quit application", 
				styles.KeyStyle("q"), "👋"),
		}
	}
	
	menu := menuStyle.Render(strings.Join(menuItems, "\n"))
	
	// Footer with responsive content
	var footerText string
	if m.width < 100 {
		// Compact footer for small terminals
		footerText = fmt.Sprintf("localhost:%s • %d profiles", m.port, len(m.profiles))
	} else {
		// Full footer
		footerText = fmt.Sprintf("🌍 Listening on localhost:%s • ⚡ Ready to serve TLS fingerprinted requests • 🔒 %d profiles available", m.port, len(m.profiles))
	}
	
	footer := styles.StatusBarStyle(m.width).
		MarginTop(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.BorderMuted).
		PaddingTop(1).
		Render(footerText)
	
	// Assemble the view
	parts := []string{title}
	
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	
	parts = append(parts, example, menu, footer)
	
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderParamBrowser renders the parameter browser view
func (m *InteractiveApp) renderParamBrowser() string {
	return m.paramBrowserModel.View()
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
	titleStyle := styles.TitleStyle.Copy().
		Width(m.width)
	
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(1)
	
	return titleStyle.Render(title) + "\n" + subtitleStyle.Render(subtitle)
}

// renderGoodbye renders the goodbye message
func (m *InteractiveApp) renderGoodbye() string {
	style := styles.TitleStyle.Copy().
		Width(m.width).
		MarginTop(2)
	
	return style.Render("Thank you for using CycleTLS-Proxy! 👋")
}


// ShouldStartServer returns true if the user chose to start the server
func (m *InteractiveApp) ShouldStartServer() bool {
	return m.startServer
}

// IsServerMode returns true if the app should run in server mode
func (m *InteractiveApp) IsServerMode() bool {
	return m.serverMode
}