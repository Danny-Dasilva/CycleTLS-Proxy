// param_browser.go - Split-pane parameter browser for CycleTLS-Proxy
package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/cmd/proxy/styles"
)

// Parameter represents a configurable parameter
type Parameter struct {
	Name            string
	Required        bool
	Desc            string
	Example         string
	CurlExample     string
	Category        string
	Complexity      string // "green", "yellow", "blue"
	ComplexityLevel int    // 1=green, 2=yellow, 3=blue
}

// FilterValue implements list.Item interface
func (p Parameter) FilterValue() string {
	return p.Name + " " + p.Desc
}

// Title returns the parameter name for list display
func (p Parameter) Title() string {
	if p.Required {
		return fmt.Sprintf("🔴 %s (REQUIRED)", p.Name)
	}
	return fmt.Sprintf("🔹 %s", p.Name)
}

// Description returns the parameter description for list display
func (p Parameter) Description() string {
	var complexityIcon string
	switch p.Complexity {
	case "green":
		complexityIcon = "🟢"
	case "yellow":
		complexityIcon = "🟡"
	case "blue":
		complexityIcon = "🔵"
	default:
		complexityIcon = "⚪"
	}
	return fmt.Sprintf("%s %s", p.Desc, complexityIcon)
}

// ParamBrowserModel represents the split-pane parameter browser
type ParamBrowserModel struct {
	list       list.Model
	viewport   viewport.Model
	width      int
	height     int
	focused    int // 0 = list, 1 = viewport
	port       string
	ready      bool
	statusMsg  string // Status message for user feedback
	statusTime int64  // Timestamp for status message expiry
}

// LegendItem represents the complexity legend as a non-selectable list item
type LegendItem struct{}

// FilterValue implements list.Item interface
func (l LegendItem) FilterValue() string {
	return "legend complexity guide"
}

// Title returns the legend title
func (l LegendItem) Title() string {
	return "🟢 Basic  🟡 Intermediate  🔵 Advanced"
}

// Description returns the legend description
func (l LegendItem) Description() string {
	return "Complexity guide for all parameters below"
}

// NewParamBrowserModel creates a new parameter browser model
func NewParamBrowserModel(port string) ParamBrowserModel {
	// Define all parameters
	parameters := []list.Item{
		// Add complexity legend as first item
		LegendItem{},

		// Required proxy headers
		Parameter{
			Name:            "X-URL",
			Required:        true,
			Desc:            "Target URL to proxy the request to",
			Example:         "https://httpbin.org/ip",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" http://localhost:%s`, port),
			Category:        "Required",
			Complexity:      "green",
			ComplexityLevel: 1,
		},

		// Basic proxy headers
		Parameter{
			Name:            "X-IDENTIFIER",
			Required:        false,
			Desc:            "Browser profile for TLS fingerprinting",
			Example:         "chrome, firefox, safari, edge",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-IDENTIFIER: firefox" http://localhost:%s`, port),
			Category:        "Basic",
			Complexity:      "green",
			ComplexityLevel: 1,
		},
		Parameter{
			Name:            "X-SESSION-ID",
			Required:        false,
			Desc:            "Session identifier for connection reuse",
			Example:         "my-session-123",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://api.example.com" -H "X-SESSION-ID: my-session-123" http://localhost:%s`, port),
			Category:        "Basic",
			Complexity:      "green",
			ComplexityLevel: 1,
		},
		Parameter{
			Name:            "X-TIMEOUT",
			Required:        false,
			Desc:            "Custom timeout in seconds (1-300)",
			Example:         "30",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/delay/10" -H "X-TIMEOUT: 15" http://localhost:%s`, port),
			Category:        "Basic",
			Complexity:      "green",
			ComplexityLevel: 1,
		},
		Parameter{
			Name:            "X-PROXY",
			Required:        false,
			Desc:            "Upstream proxy server configuration",
			Example:         "http://user:pass@proxy.example.com:8080",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-PROXY: http://user:pass@proxy:8080" http://localhost:%s`, port),
			Category:        "Basic",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},

		// Advanced TLS parameters
		Parameter{
			Name:            "X-JA3",
			Required:        false,
			Desc:            "Custom JA3 TLS fingerprint string",
			Example:         "771,4865-4867-4866-49195-49199...",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-JA3: 771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-156-157-47-53" http://localhost:%s`, port),
			Category:        "Advanced TLS",
			Complexity:      "blue",
			ComplexityLevel: 3,
		},
		Parameter{
			Name:            "X-JA4",
			Required:        false,
			Desc:            "JA4 enhanced TLS fingerprinting token",
			Example:         "t13d1516h2_8daaf6152771_02713d6af862",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-JA4: t13d1516h2_8daaf6152771_02713d6af862" http://localhost:%s`, port),
			Category:        "Advanced TLS",
			Complexity:      "blue",
			ComplexityLevel: 3,
		},
		Parameter{
			Name:            "X-HTTP2-FINGERPRINT",
			Required:        false,
			Desc:            "HTTP/2 specific fingerprint for connection settings",
			Example:         "1:65536;2:0;4:131072;5:16384|15663105|0|m,a,s,p",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-HTTP2-FINGERPRINT: 1:65536;2:0;4:131072;5:16384|15663105|0|m,a,s,p" http://localhost:%s`, port),
			Category:        "Advanced TLS",
			Complexity:      "blue",
			ComplexityLevel: 3,
		},
		Parameter{
			Name:            "X-USER-AGENT",
			Required:        false,
			Desc:            "Custom user agent string override",
			Example:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/user-agent" -H "X-USER-AGENT: Mozilla/5.0 (custom)" http://localhost:%s`, port),
			Category:        "Advanced TLS",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},

		// Connection control
		Parameter{
			Name:            "X-HEADER-ORDER",
			Required:        false,
			Desc:            "Custom header ordering for fingerprinting",
			Example:         "accept,user-agent,accept-encoding,accept-language",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/headers" -H "X-HEADER-ORDER: accept,user-agent,accept-encoding" http://localhost:%s`, port),
			Category:        "Connection",
			Complexity:      "blue",
			ComplexityLevel: 3,
		},
		Parameter{
			Name:            "X-INSECURE",
			Required:        false,
			Desc:            "Skip TLS certificate verification",
			Example:         "true",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://self-signed.badssl.com" -H "X-INSECURE: true" http://localhost:%s`, port),
			Category:        "Connection",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},
		Parameter{
			Name:            "X-FORCE-HTTP1",
			Required:        false,
			Desc:            "Force HTTP/1.1 protocol usage",
			Example:         "true",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-FORCE-HTTP1: true" http://localhost:%s`, port),
			Category:        "Connection",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},
		Parameter{
			Name:            "X-FORCE-HTTP3",
			Required:        false,
			Desc:            "Force HTTP/3/QUIC protocol usage",
			Example:         "true",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-FORCE-HTTP3: true" http://localhost:%s`, port),
			Category:        "Connection",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},
		Parameter{
			Name:            "X-ENABLE-CONNECTION-REUSE",
			Required:        false,
			Desc:            "Enable TCP connection reuse for performance",
			Example:         "true",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-ENABLE-CONNECTION-REUSE: true" http://localhost:%s`, port),
			Category:        "Connection",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},
		Parameter{
			Name:            "X-TLS13-AUTORETRY",
			Required:        false,
			Desc:            "Automatically retry with TLS 1.3 compatible curves",
			Example:         "true",
			CurlExample:     fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-TLS13-AUTORETRY: true" http://localhost:%s`, port),
			Category:        "Connection",
			Complexity:      "yellow",
			ComplexityLevel: 2,
		},
	}

	// Create list with custom delegate for better formatting
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = styles.SelectedItemStyle.Bold(true)
	delegate.Styles.NormalTitle = styles.UnselectedItemStyle
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(styles.TextSecondary)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(styles.TextMuted)
	delegate.SetHeight(2) // Show both title and description
	delegate.SetSpacing(1)
	delegate.ShowDescription = true // Explicitly enable descriptions

	paramList := list.New(parameters, delegate, 0, 0)
	paramList.SetShowHelp(false)
	paramList.SetShowStatusBar(false)
	paramList.SetFilteringEnabled(true)
	paramList.SetShowTitle(true) // Explicitly ensure title is always shown
	paramList.Title = "📋 Parameters"
	
	// Start with the second item selected (skip legend item)
	if len(parameters) > 1 {
		paramList.Select(1) // Select X-URL (first real parameter)
	}

	return ParamBrowserModel{
		list: paramList,
		port: port,
	}
}

// Init initializes the parameter browser model
func (m ParamBrowserModel) Init() tea.Cmd {
	// Initialize clipboard
	clipboard.Init()
	return nil
}

// Update handles messages for the parameter browser model
func (m ParamBrowserModel) Update(msg tea.Msg) (ParamBrowserModel, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle special cases first following Charmbracelet pattern
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle mouse events for list interaction
		if m.ready && m.focused == 0 {
			// Forward mouse events to list for click/hover functionality
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)

			// Update viewport content when mouse interaction changes selection
			m.viewport.SetContent(m.getDetailContent())
		} else if m.ready && m.focused == 1 {
			// Forward mouse events to viewport for scrolling
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Simple height calculation (match View method)
		panelHeight := msg.Height - 8
		if panelHeight < 20 {
			panelHeight = 20
		}

		leftWidth := (msg.Width - 6) / 2
		rightWidth := msg.Width - leftWidth - 4

		// Ensure minimum dimensions
		if leftWidth < 30 {
			leftWidth = 30
		}
		if rightWidth < 40 {
			rightWidth = 40
		}

		// Update list dimensions
		m.list.SetSize(leftWidth, panelHeight)

		// Initialize or update viewport
		if !m.ready {
			m.viewport = viewport.New(rightWidth, panelHeight)
			m.viewport.YPosition = 0
			m.viewport.SetContent(m.getDetailContent())
			m.ready = true
		} else {
			m.viewport.Width = rightWidth
			m.viewport.Height = panelHeight
		}

	case tea.KeyMsg:
		// Initialize on first key press if not ready
		if !m.ready && m.width > 0 && m.height > 0 {
			panelHeight := m.height - 8
			if panelHeight < 20 {
				panelHeight = 20
			}

			leftWidth := (m.width - 6) / 2
			rightWidth := m.width - leftWidth - 4

			if leftWidth < 30 {
				leftWidth = 30
			}
			if rightWidth < 40 {
				rightWidth = 40
			}

			m.viewport = viewport.New(rightWidth, panelHeight)
			m.viewport.SetContent(m.getDetailContent())
			m.list.SetSize(leftWidth, panelHeight)
			m.ready = true
		}

		// Handle non-navigation keys first that shouldn't be forwarded
		switch msg.String() {
		case "tab":
			// Toggle focus between list and viewport
			if m.focused == 0 {
				m.focused = 1
			} else {
				m.focused = 0
			}
			return m, nil
		case "c":
			// Copy current curl example to clipboard
			if m.ready {
				if err := m.copyCurrentExample(); err == nil {
					m.statusMsg = "💻 Copied curl example to clipboard"
				} else {
					m.statusMsg = fmt.Sprintf("❌ Failed to copy: %v", err)
				}
				m.statusTime = time.Now().Unix()
			}
			return m, nil
		}

		// Forward ALL other keys (including navigation) to focused component
		if !m.ready {
			return m, nil
		}

		if m.focused == 0 {
			// Handle navigation keys to skip legend item
			key := msg.String()
			currentIndex := m.list.Index()
			
			// Handle special navigation cases for legend item
			if key == "up" || key == "k" {
				if currentIndex == 1 {
					// Already at first real parameter, don't go to legend
					return m, nil
				}
			} else if key == "home" {
				// Home should go to first real parameter, not legend
				m.list.Select(1)
				m.viewport.SetContent(m.getDetailContent())
				return m, nil
			}
			
			// Forward to list (handles up/down/navigation)
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
			
			// Ensure we never land on legend item
			if m.list.Index() == 0 {
				m.list.Select(1) // Jump to first real parameter
			}

			// Update viewport content when selection might have changed
			// Check for any navigation or selection keys
			if key == "up" || key == "down" || key == "j" || key == "k" ||
				key == "enter" || key == "pgup" || key == "pgdown" ||
				key == "home" || key == "end" {
				m.viewport.SetContent(m.getDetailContent())
			}
		} else {
			// Forward to viewport (handles scrolling)
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Clear status message after 3 seconds
	if m.statusMsg != "" && time.Now().Unix()-m.statusTime > 3 {
		m.statusMsg = ""
		m.statusTime = 0
	}

	return m, tea.Batch(cmds...)
}

// View renders the parameter browser
func (m ParamBrowserModel) View() string {
	if m.width == 0 {
		return "Initializing parameter browser..."
	}

	// If not ready, show a simple layout
	if !m.ready {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("📋 Parameter Browser\n\nPress any key to initialize...")
	}

	// Build the complete view
	var parts []string

	// Simple height calculation to restore border visibility
	panelHeight := m.height - 8 // Simple approach that worked before

	// Find sweet spot for legend visibility without breaking borders
	if panelHeight < 20 {
		panelHeight = 20
	}

	// Calculate panel dimensions with proper border accounting
	// Total overhead: borders (4 chars) + gap (2 chars) = 6 chars
	leftPanelWidth := (m.width - 6) / 2
	rightPanelWidth := m.width - leftPanelWidth - 4 // Precise remainder calculation

	// Ensure minimum widths
	if leftPanelWidth < 30 {
		leftPanelWidth = 30
	}
	if rightPanelWidth < 40 {
		rightPanelWidth = 40
	}

	// Create panel styles with proper borders
	leftStyle := lipgloss.NewStyle().
		Width(leftPanelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderUnfocused).
		Padding(1)

	rightStyle := lipgloss.NewStyle().
		Width(rightPanelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderUnfocused).
		Padding(1)

	// Apply focus styling
	if m.focused == 0 {
		leftStyle = leftStyle.BorderForeground(styles.BorderFocused)
	} else {
		rightStyle = rightStyle.BorderForeground(styles.BorderFocused)
	}

	// Render panels
	leftPanel := leftStyle.Render(m.list.View())
	rightPanel := rightStyle.Render(m.renderDetailView())

	// Join panels horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	parts = append(parts, content)

	// Footer with top spacing
	footer := m.renderFooter()
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderHeader renders the browser header
func (m ParamBrowserModel) renderHeader() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.AccentGreen).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(1)

	title := titleStyle.Render("⚙️ Parameter Browser - Configure your CycleTLS requests")

	// Complexity legend
	complexityStyle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(2)

	complexityLegend := complexityStyle.Render("🟢 Basic  🟡 Intermediate  🔵 Advanced")

	return lipgloss.JoinVertical(lipgloss.Left, title, complexityLegend)
}

// renderFooter renders the browser footer
func (m ParamBrowserModel) renderFooter() string {
	// Show status message if present, otherwise show help
	if m.statusMsg != "" {
		statusStyle := styles.StatusBarStyle(m.width).
			MarginTop(1).
			Foreground(styles.AccentGreen)
		return statusStyle.Render(m.statusMsg)
	}

	var keys []string

	if m.focused == 0 {
		keys = append(keys,
			styles.KeyStyle("↑/↓")+"Navigate",
			styles.KeyStyle("tab")+"Switch panels",
			styles.KeyStyle("/")+"Filter",
			styles.KeyStyle("c")+"Copy example",
		)
	} else {
		keys = append(keys,
			styles.KeyStyle("↑/↓")+"Scroll",
			styles.KeyStyle("tab")+"Switch panels",
		)
	}

	keys = append(keys,
		styles.KeyStyle("esc")+"Back to menu",
	)

	footerStyle := styles.StatusBarStyle(m.width).
		MarginTop(1)

	return footerStyle.Render(strings.Join(keys, " • "))
}

// renderDetailView renders the right panel with parameter details
func (m ParamBrowserModel) renderDetailView() string {
	return m.viewport.View()
}

// getDetailContent generates the detail content for the selected parameter
func (m ParamBrowserModel) getDetailContent() string {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return "Select a parameter to see details..."
	}

	// Handle legend item selection
	if _, ok := selectedItem.(LegendItem); ok {
		return m.getLegendDetailContent()
	}

	param, ok := selectedItem.(Parameter)
	if !ok {
		return "Error loading parameter details"
	}

	var content strings.Builder

	// Parameter name with status
	nameStyle := styles.HeaderStyle
	if param.Required {
		nameStyle = nameStyle.Foreground(styles.AccentRed)
		content.WriteString(nameStyle.Render(fmt.Sprintf("%s (REQUIRED)", param.Name)))
	} else {
		nameStyle = nameStyle.Foreground(styles.AccentGreen)
		content.WriteString(nameStyle.Render(fmt.Sprintf("%s (optional)", param.Name)))
	}
	content.WriteString("\n\n")

	// Complexity indicator
	var complexityText, complexityIcon string
	switch param.Complexity {
	case "green":
		complexityIcon = "🟢"
		complexityText = "Basic"
	case "yellow":
		complexityIcon = "🟡"
		complexityText = "Intermediate"
	case "blue":
		complexityIcon = "🔵"
		complexityText = "Advanced"
	default:
		complexityIcon = "⚪"
		complexityText = "Unknown"
	}

	complexityStyle := styles.InfoStyle.MarginBottom(1)
	content.WriteString(complexityStyle.Render(fmt.Sprintf("Complexity: %s %s", complexityIcon, complexityText)))
	content.WriteString("\n\n")

	// Description
	descStyle := styles.ContentStyle
	content.WriteString(descStyle.Render(param.Desc))
	content.WriteString("\n\n")

	// Example value
	if param.Example != "" {
		exampleHeaderStyle := styles.WarningStyle
		content.WriteString(exampleHeaderStyle.Render("📝 Example Value:"))
		content.WriteString("\n")

		exampleStyle := styles.CodeStyle.MarginLeft(0)
		content.WriteString(exampleStyle.Render(param.Example))
		content.WriteString("\n\n")
	}

	// Curl example
	if param.CurlExample != "" {
		curlHeaderStyle := styles.SuccessStyle
		content.WriteString(curlHeaderStyle.Render("💻 Example Usage:"))
		content.WriteString("\n")

		// Format curl command like in profile browser
		curlStyle := styles.CodeStyle.
			Background(lipgloss.Color("#2D3748")).
			Padding(1).
			MarginTop(1).
			Width(m.viewport.Width - 4)

		// Format with backslash continuation for readability
		formattedCmd := m.formatCurlCommand(param.CurlExample)
		content.WriteString(curlStyle.Render(formattedCmd))
		content.WriteString("\n\n")
	}

	// Usage notes based on parameter type
	content.WriteString(m.getUsageNotes(param))

	return content.String()
}

// getLegendDetailContent generates content for the legend item
func (m ParamBrowserModel) getLegendDetailContent() string {
	var content strings.Builder

	// Header
	headerStyle := styles.HeaderStyle.Foreground(styles.AccentPurple)
	content.WriteString(headerStyle.Render("🏷️ Parameter Complexity Guide"))
	content.WriteString("\n\n")

	// Complexity explanations
	legendItems := []struct {
		icon  string
		level string
		desc  string
	}{
		{"🟢", "Basic", "Easy to use parameters that require minimal knowledge. Safe for beginners."},
		{"🟡", "Intermediate", "Moderate complexity parameters that may require some understanding of networking or TLS concepts."},
		{"🔵", "Advanced", "Complex parameters for experts. Requires deep knowledge of TLS, HTTP protocols, or fingerprinting techniques."},
	}

	for _, item := range legendItems {
		// Level header
		levelStyle := styles.SuccessStyle.Foreground(styles.AccentGreen)
		content.WriteString(levelStyle.Render(fmt.Sprintf("%s %s", item.icon, item.level)))
		content.WriteString("\n")

		// Description
		descStyle := styles.ContentStyle.MarginLeft(0).MarginBottom(2)
		content.WriteString(descStyle.Render(item.desc))
		content.WriteString("\n\n")
	}

	// Usage tip
	tipStyle := styles.InfoStyle.Foreground(styles.TextMuted)
	content.WriteString(tipStyle.Render("💡 Tip: Start with Basic (🟢) parameters, then progress to more advanced options as needed."))

	return content.String()
}

// formatCurlCommand formats long curl commands for better display
func (m ParamBrowserModel) formatCurlCommand(curlCmd string) string {
	// Split on headers to make it more readable
	parts := strings.Split(curlCmd, " -H ")
	if len(parts) == 1 {
		// No headers, return as is
		return curlCmd
	}

	// Format with backslash continuation
	var result strings.Builder
	result.WriteString(parts[0])

	for i := 1; i < len(parts); i++ {
		result.WriteString(" \\\n     -H ")
		result.WriteString(parts[i])
	}

	return result.String()
}

// getUsageNotes provides parameter-specific usage guidance
func (m ParamBrowserModel) getUsageNotes(param Parameter) string {
	var notes strings.Builder

	noteStyle := styles.ContentStyle.Foreground(styles.TextMuted)

	switch param.Name {
	case "X-URL":
		notes.WriteString(noteStyle.Render("💬 This parameter is required for all requests. Must be a valid HTTP/HTTPS URL."))
	case "X-IDENTIFIER":
		notes.WriteString(noteStyle.Render("💬 Available profiles: chrome138, chrome139, chrome, chrome_windows, firefox, firefox_windows, safari_ios, safari, edge, okhttp, chrome_legacy_tls12"))
	case "X-SESSION-ID":
		notes.WriteString(noteStyle.Render("💬 Use the same session ID across multiple requests to enable connection reuse and cookie persistence."))
	case "X-JA3":
		notes.WriteString(noteStyle.Render("💬 🔵 Advanced users only. Custom JA3 fingerprint overrides the selected browser profile. Requires deep TLS knowledge."))
	case "X-JA4":
		notes.WriteString(noteStyle.Render("💬 🔵 Advanced TLS fingerprinting. More sophisticated than JA3, includes additional connection details."))
	case "X-HTTP2-FINGERPRINT":
		notes.WriteString(noteStyle.Render("💬 🔵 Expert level parameter. Controls HTTP/2 connection settings for advanced fingerprint evasion."))
	case "X-HEADER-ORDER":
		notes.WriteString(noteStyle.Render("💬 🔵 Advanced fingerprinting bypass. Specify exact header ordering to mimic specific browsers."))
	case "X-TIMEOUT":
		notes.WriteString(noteStyle.Render("💬 Value must be between 1-300 seconds. Default is 30 seconds."))
	case "X-PROXY":
		notes.WriteString(noteStyle.Render("💬 🟡 Supports HTTP, SOCKS4, SOCKS5 proxies. Format: protocol://[user:pass@]host:port"))
	case "X-USER-AGENT":
		notes.WriteString(noteStyle.Render("💬 🟡 Override the user agent string. Use with caution as it may conflict with TLS fingerprints."))
	case "X-INSECURE":
		notes.WriteString(noteStyle.Render("💬 🟡 Disables TLS certificate verification. Use only for testing with self-signed certificates."))
	case "X-FORCE-HTTP1":
		notes.WriteString(noteStyle.Render("💬 🟡 Forces HTTP/1.1 protocol. Useful for testing or bypassing HTTP/2-specific issues."))
	case "X-FORCE-HTTP3":
		notes.WriteString(noteStyle.Render("💬 🟡 Forces HTTP/3 (QUIC) protocol. Experimental feature, may not work with all servers."))
	case "X-ENABLE-CONNECTION-REUSE":
		notes.WriteString(noteStyle.Render("💬 🟡 Enables TCP connection reuse for better performance across multiple requests."))
	case "X-TLS13-AUTORETRY":
		notes.WriteString(noteStyle.Render("💬 🟡 Automatically retries connections with TLS 1.3 compatible curves when initial connection fails. Improves compatibility with TLS 1.3 servers."))
	default:
		notes.WriteString(noteStyle.Render("💬 This parameter modifies the TLS/HTTP behavior of your requests."))
	}

	return notes.String()
}

// copyCurrentExample copies the current parameter's curl example to clipboard
func (m ParamBrowserModel) copyCurrentExample() error {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return fmt.Errorf("no item selected")
	}

	param, ok := selectedItem.(Parameter)
	if !ok {
		return fmt.Errorf("invalid item type")
	}

	// Copy the curl example to clipboard
	clipboard.Write(clipboard.FmtText, []byte(param.CurlExample))
	return nil
}
