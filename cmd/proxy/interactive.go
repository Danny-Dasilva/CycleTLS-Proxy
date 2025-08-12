// interactive.go - Interactive Bubble Tea application for CycleTLS-Proxy
package main

import (
	"fmt"
	"strings"
	"time"

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
	ModeAddProfile
	ModeServer
	ModeConfig
	ModeMonitor
	ModeTest
)

// RequestLogMsg is an alias for the proxy RequestLogEntry
type RequestLogMsg = proxy.RequestLogEntry

// MonitorEventMsg wraps proxy monitor events
type MonitorEventMsg = proxy.MonitorEvent

// tickMsg is sent periodically to update the display
type tickMsg time.Time

// InteractiveApp is the main Bubble Tea model for the interactive application
type InteractiveApp struct {
	mode     AppMode
	width    int
	height   int
	logger   *log.Logger
	port     string
	profiles map[string]fingerprints.Profile

	// Sub-models
	paramBrowserModel models.ParamBrowserModel
	profileModel      models.ProfileModel
	addProfileModel   models.AddProfileModel
	configModel       models.ConfigModel
	monitorModel      models.MonitorModel
	logo              *models.GradientLogo

	// Live request logging and monitoring
	requestLogs    []proxy.RequestLogEntry
	maxLogs        int
	logChannel     chan proxy.RequestLogEntry
	monitorChannel chan proxy.MonitorEvent
	proxyHandler   *proxy.Handler

	// State
	quitting    bool
	startServer bool
	serverMode  bool
}

// NewInteractiveApp creates a new interactive application
func NewInteractiveApp(port string, logger *log.Logger, logChannel chan proxy.RequestLogEntry, monitorChannel chan proxy.MonitorEvent, handler ...*proxy.Handler) *InteractiveApp {
	profiles := fingerprints.GetDefaultProfiles()

	// Create monitor model with or without handler reference
	var monitorModel models.MonitorModel
	var rotator *fingerprints.ProfileRotator
	if len(handler) > 0 && handler[0] != nil {
		monitorModel = models.NewMonitorModelWithHandler(handler[0], monitorChannel)
		rotator = handler[0].GetRotator()
	} else {
		monitorModel = models.NewMonitorModel()
		// For non-server mode, create a default rotator
		rotator = fingerprints.NewProfileRotator(nil)
	}

	app := &InteractiveApp{
		mode:     ModeMenu,
		logger:   logger,
		port:     port,
		profiles: profiles,

		paramBrowserModel: models.NewParamBrowserModel(port),
		profileModel:      models.NewProfileModel(profiles, rotator),
		addProfileModel:   models.NewAddProfileModel(),
		configModel:       models.NewConfigModel(),
		monitorModel:      monitorModel,
		logo:              models.NewGradientLogo(0, 0),

		// Initialize request logging and monitoring
		requestLogs:    make([]proxy.RequestLogEntry, 0, 30),
		maxLogs:        30,
		logChannel:     logChannel,
		monitorChannel: monitorChannel,
	}

	// If handler is provided, we're in server mode
	if len(handler) > 0 && handler[0] != nil {
		app.serverMode = true
		app.proxyHandler = handler[0]

		// Start periodic metrics updates from handler
		handler[0].StartPeriodicMetricsUpdates(2 * time.Second)
	}

	return app
}

// Init initializes the Bubble Tea application
func (m *InteractiveApp) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.EnterAltScreen,
		tea.EnableMouseAllMotion,
		m.listenForRequestLogs(),
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	}

	// Add monitor event listener if available
	if m.monitorChannel != nil {
		cmds = append(cmds, m.listenForMonitorEvents())
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the model state
func (m *InteractiveApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle WindowSizeMsg specially to update all models
	switch msg := msg.(type) {
	case proxy.RequestLogEntry:
		// Add new request log
		m.addRequestLog(msg)
		return m, nil

	case proxy.MonitorEvent:
		// Forward monitor events to the monitor model
		var cmd tea.Cmd
		m.monitorModel, cmd = m.monitorModel.Update(models.MonitorEventMsg(msg))
		return m, cmd

	case tickMsg:
		// Continue listening for logs and periodic updates
		cmds := []tea.Cmd{
			m.listenForRequestLogs(),
			tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		}

		// Continue listening for monitor events
		if m.monitorChannel != nil {
			cmds = append(cmds, m.listenForMonitorEvents())
		}

		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update logo dimensions
		if m.logo != nil {
			m.logo = models.NewGradientLogo(msg.Width, msg.Height)
		}

		// Forward WindowSizeMsg to all sub-models
		var cmds []tea.Cmd

		var cmd tea.Cmd
		m.paramBrowserModel, cmd = m.paramBrowserModel.Update(msg)
		cmds = append(cmds, cmd)

		m.profileModel, cmd = m.profileModel.Update(msg)
		cmds = append(cmds, cmd)

		m.addProfileModel, cmd = m.addProfileModel.Update(msg)
		cmds = append(cmds, cmd)

		m.configModel, cmd = m.configModel.Update(msg)
		cmds = append(cmds, cmd)

		m.monitorModel, cmd = m.monitorModel.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Handle only global keys that affect mode switching
		key := msg.String()

		// Global keys that work in any mode
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

		// Menu-specific keys (only when in menu mode)
		if m.mode == ModeMenu {
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
			// If no menu key matched, fall through to delegation
		}
		// Fall through to delegate to active sub-model
	}

	// Delegate ALL messages (including unhandled KeyMsg and MouseMsg) to sub-models based on current mode
	switch m.mode {
	case ModeParamBrowser:
		var cmd tea.Cmd
		m.paramBrowserModel, cmd = m.paramBrowserModel.Update(msg)
		return m, cmd

	case ModeProfiles:
		// Check for switch to add profile message
		if _, ok := msg.(models.SwitchToAddProfileMsg); ok {
			m.mode = ModeAddProfile
			return m, nil
		}
		
		var cmd tea.Cmd
		m.profileModel, cmd = m.profileModel.Update(msg)
		return m, cmd

	case ModeAddProfile:
		var cmd tea.Cmd
		m.addProfileModel, cmd = m.addProfileModel.Update(msg)
		return m, cmd

	case ModeConfig:
		var cmd tea.Cmd
		m.configModel, cmd = m.configModel.Update(msg)
		return m, cmd

	case ModeMonitor:
		var cmd tea.Cmd
		m.monitorModel, cmd = m.monitorModel.Update(msg)
		return m, cmd

	case ModeTest:
		// Test mode doesn't have a model yet
		return m, nil

	case ModeMenu:
		// Menu mode doesn't need to forward messages
		return m, nil
	}

	return m, nil
}

// listenForRequestLogs creates a command that listens for incoming request logs
func (m *InteractiveApp) listenForRequestLogs() tea.Cmd {
	if m.logChannel == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case log := <-m.logChannel:
			return log
		default:
			// Non-blocking read - return nil if no logs available
			return nil
		}
	}
}

// addRequestLog adds a new request log to the buffer
func (m *InteractiveApp) addRequestLog(log proxy.RequestLogEntry) {
	// Add to the beginning of the slice (newest first)
	m.requestLogs = append([]proxy.RequestLogEntry{log}, m.requestLogs...)

	// Keep only the most recent logs
	if len(m.requestLogs) > m.maxLogs {
		m.requestLogs = m.requestLogs[:m.maxLogs]
	}
}

// renderRequestLogs renders the live request logs panel with header (for monitor mode)
func (m *InteractiveApp) renderRequestLogs() string {
	var content strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(styles.AccentPurple).
		Bold(true).
		MarginBottom(1)

	content.WriteString(headerStyle.Render("🕐 Live Request Logs"))
	content.WriteString("\n\n")

	// Add the actual logs content
	logsContent := m.renderRequestLogsContent()
	content.WriteString(logsContent)

	return content.String()
}

// renderRequestLogsContent renders just the request logs content without header
func (m *InteractiveApp) renderRequestLogsContent() string {
	var content strings.Builder

	if len(m.requestLogs) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.TextMuted).
			Italic(true)

		content.WriteString(emptyStyle.Render("Waiting for requests..."))
		return content.String()
	}

	// Display recent requests (up to 20 most recent)
	maxDisplay := 20
	if len(m.requestLogs) < maxDisplay {
		maxDisplay = len(m.requestLogs)
	}

	for i := 0; i < maxDisplay; i++ {
		req := m.requestLogs[i]

		// Format timestamp
		timeStr := req.Timestamp.Format("15:04:05")

		// Format method with fixed width
		methodStr := fmt.Sprintf("%-6s", req.Method)

		// Format status with color coding
		statusStr := fmt.Sprintf("%d", req.Status)
		var statusColor lipgloss.Color
		if req.Status >= 200 && req.Status < 300 {
			statusColor = styles.AccentGreen // Green for 2xx
		} else if req.Status >= 300 && req.Status < 400 {
			statusColor = styles.AccentOrange // Orange for 3xx
		} else if req.Status >= 400 && req.Status < 500 {
			statusColor = styles.AccentOrange // Orange for 4xx
		} else if req.Status >= 500 {
			statusColor = styles.AccentRed // Red for 5xx
		} else {
			statusColor = styles.TextMuted // Gray for others
		}

		// Format URL (truncate if too long)
		urlStr := req.URL
		maxURLLen := 35 // Adjust based on right panel width
		if len(urlStr) > maxURLLen {
			urlStr = urlStr[:maxURLLen-3] + "..."
		}

		// Format duration
		durationStr := ""
		if req.Duration > 0 {
			if req.Duration < time.Second {
				durationStr = fmt.Sprintf("%.0fms", float64(req.Duration.Nanoseconds())/1000000)
			} else {
				durationStr = fmt.Sprintf("%.1fs", req.Duration.Seconds())
			}
		}

		// Create the log entry
		timeStyled := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(timeStr)
		methodStyled := lipgloss.NewStyle().Foreground(styles.PrimaryEnd).Render(methodStr)
		statusStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusStr)
		urlStyled := lipgloss.NewStyle().Foreground(styles.TextPrimary).Render(urlStr)
		durationStyled := lipgloss.NewStyle().Foreground(styles.TextSecondary).Render(durationStr)

		// Build the log line
		var logLine string
		if durationStr != "" {
			logLine = fmt.Sprintf("%s %s %s %s %s", timeStyled, methodStyled, statusStyled, durationStyled, urlStyled)
		} else {
			logLine = fmt.Sprintf("%s %s %s %s", timeStyled, methodStyled, statusStyled, urlStyled)
		}

		content.WriteString(logLine)
		content.WriteString("\n")
	}

	// Footer with log count
	if len(m.requestLogs) > maxDisplay {
		footerStyle := lipgloss.NewStyle().
			Foreground(styles.TextMuted).
			Italic(true).
			MarginTop(1)

		footer := footerStyle.Render(fmt.Sprintf("... and %d more", len(m.requestLogs)-maxDisplay))
		content.WriteString(footer)
	}

	return content.String()
}

// listenForMonitorEvents creates a command that listens for monitor events
func (m *InteractiveApp) listenForMonitorEvents() tea.Cmd {
	if m.monitorChannel == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case event := <-m.monitorChannel:
			return event
		default:
			// Non-blocking read - return nil if no events available
			return nil
		}
	}
}

// Note: handleKeyPress and handleMenuKeys have been integrated into Update method
// to properly forward messages to sub-models

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
	case ModeAddProfile:
		return m.renderAddProfile()
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

// renderMenu renders the main menu with smooth responsive layout
func (m *InteractiveApp) renderMenu() string {
	// Use a smooth responsive approach based on available width
	return m.renderResponsiveMenu()
}

// renderResponsiveMenu creates a smooth responsive layout without hard breakpoints
func (m *InteractiveApp) renderResponsiveMenu() string {
	// For very wide terminals (>180), center the content
	if m.width > 180 {
		maxContentWidth := 160
		contentWidth := maxContentWidth
		if m.width < maxContentWidth {
			contentWidth = m.width - 4
		}

		// Create main content using constrained width
		content := m.renderResponsiveContent(contentWidth)

		// Center the content horizontally
		centerStyle := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center)

		return centerStyle.Render(content)
	}

	// Use the main responsive content renderer
	return m.renderResponsiveContent(m.width - 4)
}

// renderResponsiveContent renders content with smooth responsive behavior
func (m *InteractiveApp) renderResponsiveContent(width int) string {
	var parts []string

	// Render responsive gradient logo
	var title string
	switch {
	case width < 60:
		title = m.logo.RenderTiny()
	case width < 80:
		title = m.logo.RenderCompact()
	case width < 120:
		title = m.logo.Render()
	default:
		title = m.logo.RenderWithSubtitle("Advanced TLS Fingerprint Proxy Server")
	}
	parts = append(parts, title)

	// Server status
	if m.serverMode {
		statusStyle := styles.SuccessStyle.
			Align(lipgloss.Center).
			Width(width).
			MarginBottom(1)
		statusLine := statusStyle.Render("🟢 SERVER RUNNING • Ready to accept requests")
		parts = append(parts, statusLine)
	}

	// Example and recent requests - smooth responsive behavior
	if m.serverMode && width > 90 {
		// Side-by-side layout when we have sufficient width
		sideBySideContent := m.renderQuickStartAndRecentRequests(width)
		parts = append(parts, sideBySideContent)
	} else if m.serverMode {
		// Stack vertically for narrow terminals in server mode
		example := m.renderQuickStartExample(width)
		parts = append(parts, example)

		// Add compact recent requests below
		recentRequests := m.renderCompactRecentRequestsBox(width)
		parts = append(parts, recentRequests)
	} else {
		// Just quick start example for non-server mode
		example := m.renderQuickStartExample(width)
		parts = append(parts, example)
	}

	// Menu
	menu := m.renderMenuItems(width)
	parts = append(parts, menu)

	// Footer
	footer := m.renderFooter()
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderMenuSplitLayout renders the split-pane layout for servers with live logs
func (m *InteractiveApp) renderMenuSplitLayout() string {
	// Calculate panel dimensions for split layout
	leftPanelWidth := int(float64(m.width) * 0.65)  // 65% for main content
	rightPanelWidth := m.width - leftPanelWidth - 2 // Remaining for logs + gap

	// Ensure minimum widths
	if leftPanelWidth < 60 {
		leftPanelWidth = 60
	}
	if rightPanelWidth < 40 {
		rightPanelWidth = 40
	}

	// Create left panel content
	leftPanelContent := m.renderLeftPanel(leftPanelWidth)

	// Create right panel content (live logs)
	rightPanelContent := m.renderRightPanel(rightPanelWidth)

	// Style the panels
	leftPanelStyle := styles.GetLeftPanelStyle(leftPanelWidth, m.height, false)
	rightPanelStyle := styles.GetRightPanelStyle(rightPanelWidth, m.height, false)

	leftPanel := leftPanelStyle.Render(leftPanelContent)
	rightPanel := rightPanelStyle.Render(rightPanelContent)

	// Join panels horizontally with a small gap
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
}

// renderMenuCentered renders the menu with centered content for ultra-wide terminals
func (m *InteractiveApp) renderMenuCentered() string {
	// Define maximum content width for ultra-wide terminals
	maxContentWidth := 160
	contentWidth := maxContentWidth
	if m.width < maxContentWidth {
		contentWidth = m.width - 4
	}

	// Create main content using single column logic but with constrained width
	content := m.renderCenteredContent(contentWidth)

	// Center the content horizontally in the full terminal width
	centerStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center)

	return centerStyle.Render(content)
}

// renderCenteredContent renders the main content with constrained width
func (m *InteractiveApp) renderCenteredContent(width int) string {
	var parts []string

	// Render responsive gradient logo
	var title string
	switch {
	case width < 60:
		title = m.logo.RenderTiny()
	case width < 80:
		title = m.logo.RenderCompact()
	case width < 120:
		title = m.logo.Render() // Uses responsive sizing
	default:
		title = m.logo.RenderWithSubtitle("Advanced TLsS Fingerprint Proxy Server")
	}
	parts = append(parts, title)

	// Server status
	if m.serverMode {
		statusStyle := styles.SuccessStyle.
			Align(lipgloss.Center).
			Width(width).
			MarginBottom(1)
		statusLine := statusStyle.Render("🟢 SERVER RUNNING • Ready to accept requests")
		parts = append(parts, statusLine)
	}

	// Quick start example and recent requests side-by-side (server mode)
	if m.serverMode {
		sideBySideContent := m.renderQuickStartAndRecentRequests(width)
		parts = append(parts, sideBySideContent)
	} else {
		// Just quick start example for non-server mode
		example := m.renderQuickStartExample(width)
		parts = append(parts, example)
	}

	// Menu
	menu := m.renderMenuItems(width)
	parts = append(parts, menu)

	// Footer
	footer := m.renderFooter()
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderMenuSingleColumn renders menu in single column for small terminals
func (m *InteractiveApp) renderMenuSingleColumn() string {
	// Render responsive gradient logo
	var title string
	switch {
	case m.width < 60:
		title = m.logo.RenderTiny()
	case m.width < 80:
		title = m.logo.RenderCompact()
	case m.height < 25:
		title = m.logo.Render() // Uses responsive sizing based on breakpoints
	default:
		title = m.logo.RenderWithSubtitle("Advanced TLS Fingerprint Proxy Serverfasdf")
	}

	// Server status
	var statusLine string
	if m.serverMode {
		statusStyle := styles.SuccessStyle.
			Align(lipgloss.Center).
			Width(m.width).
			MarginBottom(0)
		statusLine = statusStyle.Render("🟢 SERVER RUNNING • Ready to accept requests")
	}

	// Quick start example and recent requests side-by-side (server mode)
	var example string
	if m.serverMode {
		example = m.renderQuickStartAndRecentRequests(m.width - 4)
	} else {
		example = m.renderQuickStartExample(m.width - 4)
	}

	// Menu
	menu := m.renderMenuItems(m.width - 4)

	// Footer
	footer := m.renderFooter()

	// Assemble the view
	parts := []string{title}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, example, menu, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderLeftPanel renders the left panel content (logo, status, example, menu)
func (m *InteractiveApp) renderLeftPanel(width int) string {
	var parts []string

	// Render responsive gradient logo based on panel width
	var title string
	switch {
	case width < 60:
		title = m.logo.RenderTiny()
	case width < 80:
		title = m.logo.RenderCompact()
	case width < 120:
		title = m.logo.Render() // Uses responsive sizing
	default:
		title = m.logo.RenderWithSubtitle("Advanced TLS Fingerprint Prox1111y Server")
	}
	parts = append(parts, title)

	// Server status
	if m.serverMode {
		statusStyle := styles.SuccessStyle.
			Align(lipgloss.Center).
			Width(width).
			MarginBottom(1)
		statusLine := statusStyle.Render("🟢 SERVER RUNNING • Ready to accept requests")
		parts = append(parts, statusLine)
	}

	// Quick start example
	example := m.renderQuickStartExample(width)
	parts = append(parts, example)

	// Menu
	menu := m.renderMenuItems(width)
	parts = append(parts, menu)

	// Footer
	footer := m.renderFooter()
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRightPanel renders the right panel content (recent requests with header)
func (m *InteractiveApp) renderRightPanel(width int) string {
	var content strings.Builder

	// Panel header
	headerStyle := styles.HeaderStyle.
		Width(width - 4). // Account for panel padding
		Align(lipgloss.Center).
		MarginBottom(1)

	content.WriteString(headerStyle.Render("🕐 Recent Requests"))
	content.WriteString("\n")

	// Request logs content (without header)
	logsContent := m.renderRequestLogsContent()
	content.WriteString(logsContent)

	return content.String()
}

// renderRecentRequestsBox renders recent requests in a styled box for ultra-wide terminals
func (m *InteractiveApp) renderRecentRequestsBox(width int) string {
	// Create box style similar to other components
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentPurple).
		Padding(1, 2).
		Width(width).
		MarginTop(1).
		MarginBottom(1)

	var content strings.Builder

	// Header
	headerStyle := styles.HeaderStyle.
		Foreground(styles.AccentPurple)
	content.WriteString(headerStyle.Render("🕐 Recent Requests"))
	content.WriteString("\n\n")

	// Recent requests content
	logsContent := m.renderRequestLogsContent()
	content.WriteString(logsContent)

	return boxStyle.Render(content.String())
}

// renderQuickStartAndRecentRequests renders quick start and recent requests side-by-side
func (m *InteractiveApp) renderQuickStartAndRecentRequests(width int) string {
	// Calculate widths for two side-by-side boxes
	leftWidth := int(float64(width) * 0.6) // 60% for quick start
	rightWidth := width - leftWidth - 2    // 40% for recent requests minus gap

	// Ensure minimum widths
	if leftWidth < 40 {
		leftWidth = 40
	}
	if rightWidth < 30 {
		rightWidth = 30
	}

	// Render left panel (quick start)
	leftPanel := m.renderQuickStartExample(leftWidth)

	// Render right panel (recent requests) - compact version
	rightPanel := m.renderCompactRecentRequestsBox(rightWidth)

	// Join horizontally with a small gap
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
}

// renderCompactRecentRequestsBox renders a more compact version of recent requests
func (m *InteractiveApp) renderCompactRecentRequestsBox(width int) string {
	// Create box style similar to quick start - reduced spacing
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentPurple).
		Padding(1, 2).
		Width(width).
		MarginTop(0).
		MarginBottom(0)

	var content strings.Builder

	// Header
	headerStyle := styles.SuccessStyle
	content.WriteString(headerStyle.Render("🕐 Recent Requests:"))
	content.WriteString("\n")

	// Compact request logs content
	if len(m.requestLogs) == 0 {
		// Code style for consistency with quick start - reduced spacing
		emptyStyle := styles.CodeStyle.
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#E2E8F0")).
			Padding(1).
			MarginTop(0).
			Width(width - 6)

		content.WriteString(emptyStyle.Render("Waiting for requests..."))
	} else {
		// Show compact request logs - reduced spacing
		logsStyle := styles.CodeStyle.
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#E2E8F0")).
			Padding(1).
			MarginTop(0).
			Width(width - 6)

		var logsContent strings.Builder
		maxDisplay := 3 // Reduced from 5 to match Quick Start Example height
		if len(m.requestLogs) < maxDisplay {
			maxDisplay = len(m.requestLogs)
		}

		for i := 0; i < maxDisplay; i++ {
			req := m.requestLogs[i]
			timeStr := req.Timestamp.Format("15:04:05")
			methodStr := fmt.Sprintf("%-4s", req.Method) // Fixed width for better alignment
			statusStr := fmt.Sprintf("%d", req.Status)

			// Format URL to show meaningful part - truncate if needed
			urlStr := req.URL
			maxURLLen := width - 25 // Reserve space for time, method, status
			if len(urlStr) > maxURLLen {
				if maxURLLen > 10 {
					urlStr = urlStr[:maxURLLen-3] + "..."
				} else {
					urlStr = "..." // Very narrow, just show ellipsis
				}
			}

			// Apply color formatting like in renderRequestLogsContent
			var statusColor lipgloss.Color
			if req.Status >= 200 && req.Status < 300 {
				statusColor = styles.AccentGreen // Green for 2xx
			} else if req.Status >= 300 && req.Status < 400 {
				statusColor = styles.AccentOrange // Orange for 3xx
			} else if req.Status >= 400 && req.Status < 500 {
				statusColor = styles.AccentOrange // Orange for 4xx
			} else if req.Status >= 500 {
				statusColor = styles.AccentRed // Red for 5xx
			} else {
				statusColor = styles.TextMuted // Gray for others
			}

			// Create styled components
			timeStyled := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(timeStr)
			methodStyled := lipgloss.NewStyle().Foreground(styles.PrimaryEnd).Render(methodStr)
			statusStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusStr)
			urlStyled := lipgloss.NewStyle().Foreground(styles.TextPrimary).Render(urlStr)

			// Compact format with colored components
			logLine := fmt.Sprintf("%s %s %s %s", timeStyled, methodStyled, statusStyled, urlStyled)
			logsContent.WriteString(logLine)
			if i < maxDisplay-1 {
				logsContent.WriteString("\n")
			}
		}

		if len(m.requestLogs) > maxDisplay {
			logsContent.WriteString(fmt.Sprintf("\n... and %d more", len(m.requestLogs)-maxDisplay))
		}

		content.WriteString(logsStyle.Render(logsContent.String()))
	}

	content.WriteString("\n")

	// Add helpful note - reduced spacing
	noteStyle := styles.ContentStyle.
		Foreground(styles.TextMuted).
		MarginTop(0)

	note := "💬 Live request monitoring. Press 'm' for detailed view."
	content.WriteString(noteStyle.Render(note))

	return boxStyle.Render(content.String())
}

// renderQuickStartExample renders the quick start example in a single styled box like parameter browser
func (m *InteractiveApp) renderQuickStartExample(width int) string {
	// Create single box style matching parameter browser - reduced spacing
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentOrange).
		Padding(1, 2).
		Width(width).
		MarginTop(0).
		MarginBottom(0)

	var content strings.Builder

	// Header matching parameter browser style
	headerStyle := styles.SuccessStyle
	content.WriteString(headerStyle.Render("💻 Quick Start Example:"))
	content.WriteString("\n")

	// Curl command styled like parameter browser examples - reduced spacing
	curlStyle := styles.CodeStyle.
		Background(lipgloss.Color("#2D3748")).
		Foreground(lipgloss.Color("#E2E8F0")).
		Padding(1).
		MarginTop(0).
		Width(width - 6) // Account for box padding and borders

	// Format command with backslash continuation for readability
	var exampleCmd string
	if width < 100 {
		// Compact version for smaller terminals
		exampleCmd = fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" \
     -H "X-IDENTIFIER: chrome" \
     http://localhost:%s`, m.port)
	} else {
		// Full version with better formatting
		exampleCmd = fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" \
     -H "X-IDENTIFIER: chrome" \
     http://localhost:%s`, m.port)
	}

	content.WriteString(curlStyle.Render(exampleCmd))
	content.WriteString("\n")

	// Add helpful note matching parameter browser style - reduced spacing
	noteStyle := styles.ContentStyle.
		Foreground(styles.TextMuted).
		MarginTop(0)

	note := "💬 This example shows the basic request format. Press 'h' to browse all parameters."
	content.WriteString(noteStyle.Render(note))

	return boxStyle.Render(content.String())
}

// renderMenuItems renders the menu items box
func (m *InteractiveApp) renderMenuItems(width int) string {
	menuStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.AccentGreen).
		Padding(1, 2).
		Width(width)

	var menuItems []string
	if m.serverMode {
		menuItems = []string{
			fmt.Sprintf("%s %s Browse all parameters & examples",
				styles.KeyStyle("h"), "📋"),
			fmt.Sprintf("%s %s Browse browser profiles (%d available)",
				styles.KeyStyle("p"), "🌐", len(m.profiles)),
			fmt.Sprintf("%s %s  Configure settings",
				styles.KeyStyle("c"), "🔧"),
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
			fmt.Sprintf("%s %s  Configure settings",
				styles.KeyStyle("c"), "🔧"),
			fmt.Sprintf("%s %s Test requests",
				inactiveKeyStyle.Render(" t "), "🧪"),
			fmt.Sprintf("%s %s Monitor (requires running server)",
				inactiveKeyStyle.Render(" m "), "📊"),
			fmt.Sprintf("%s %s Quit application",
				styles.KeyStyle("q"), "👋"),
		}
	}

	return menuStyle.Render(strings.Join(menuItems, "\n"))
}

// renderFooter renders the footer
func (m *InteractiveApp) renderFooter() string {
	var footerText string
	if m.width < 100 {
		footerText = fmt.Sprintf("localhost:%s • %d profiles", m.port, len(m.profiles))
	} else {
		footerText = fmt.Sprintf("🌍 Listening on localhost:%s • ⚡ Ready to serve TLS fingerprinted requests • 🔒 %d profiles available", m.port, len(m.profiles))
	}

	return styles.StatusBarStyle(m.width).
		MarginTop(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.BorderMuted).
		PaddingTop(1).
		Render(footerText)
}

// renderParamBrowser renders the parameter browser view
func (m *InteractiveApp) renderParamBrowser() string {
	return m.paramBrowserModel.View()
}

// renderProfiles renders the profiles view
func (m *InteractiveApp) renderProfiles() string {
	return m.profileModel.View()
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
	titleStyle := styles.TitleStyle.
		Width(m.width)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(1)

	return titleStyle.Render(title) + "\n" + subtitleStyle.Render(subtitle)
}

// renderAddProfile renders the add profile form view  
func (m *InteractiveApp) renderAddProfile() string {
	return m.addProfileModel.View()
}

// renderGoodbye renders the goodbye message
func (m *InteractiveApp) renderGoodbye() string {
	style := styles.TitleStyle.
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
