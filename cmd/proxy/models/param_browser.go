// param_browser.go - Split-pane parameter browser for CycleTLS-Proxy
package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/cmd/proxy/styles"
)

// Parameter represents a configurable parameter
type Parameter struct {
	Name        string
	Required    bool
	Description string
	Example     string
	CurlExample string
	Category    string
}

// FilterValue implements list.Item interface
func (p Parameter) FilterValue() string {
	return p.Name + " " + p.Description
}

// ParamBrowserModel represents the split-pane parameter browser
type ParamBrowserModel struct {
	list     list.Model
	viewport viewport.Model
	width    int
	height   int
	focused  int // 0 = list, 1 = viewport
	port     string
	ready    bool
}

// NewParamBrowserModel creates a new parameter browser model
func NewParamBrowserModel(port string) ParamBrowserModel {
	// Define all parameters
	parameters := []list.Item{
		// Required proxy headers
		Parameter{
			Name:        "X-URL",
			Required:    true,
			Description: "Target URL to proxy the request to",
			Example:     "https://httpbin.org/ip",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" http://localhost:%s`, port),
			Category:    "Required",
		},
		
		// Basic proxy headers
		Parameter{
			Name:        "X-IDENTIFIER",
			Required:    false,
			Description: "Browser profile for TLS fingerprinting",
			Example:     "chrome, firefox, safari, edge",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-IDENTIFIER: firefox" http://localhost:%s`, port),
			Category:    "Basic",
		},
		Parameter{
			Name:        "X-SESSION-ID",
			Required:    false,
			Description: "Session identifier for connection reuse",
			Example:     "my-session-123",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://api.example.com" -H "X-SESSION-ID: my-session-123" http://localhost:%s`, port),
			Category:    "Basic",
		},
		Parameter{
			Name:        "X-TIMEOUT",
			Required:    false,
			Description: "Custom timeout in seconds (1-300)",
			Example:     "30",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/delay/10" -H "X-TIMEOUT: 15" http://localhost:%s`, port),
			Category:    "Basic",
		},
		Parameter{
			Name:        "X-PROXY",
			Required:    false,
			Description: "Upstream proxy server configuration",
			Example:     "http://user:pass@proxy.example.com:8080",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-PROXY: http://user:pass@proxy:8080" http://localhost:%s`, port),
			Category:    "Basic",
		},
		
		// Advanced TLS parameters
		Parameter{
			Name:        "X-JA3",
			Required:    false,
			Description: "Custom JA3 TLS fingerprint string",
			Example:     "771,4865-4867-4866-49195-49199...",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-JA3: 771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-156-157-47-53" http://localhost:%s`, port),
			Category:    "Advanced TLS",
		},
		Parameter{
			Name:        "X-JA4",
			Required:    false,
			Description: "JA4 enhanced TLS fingerprinting token",
			Example:     "t13d1516h2_8daaf6152771_02713d6af862",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-JA4: t13d1516h2_8daaf6152771_02713d6af862" http://localhost:%s`, port),
			Category:    "Advanced TLS",
		},
		Parameter{
			Name:        "X-HTTP2-FINGERPRINT",
			Required:    false,
			Description: "HTTP/2 specific fingerprint for connection settings",
			Example:     "1:65536;2:0;4:131072;5:16384|15663105|0|m,a,s,p",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-HTTP2-FINGERPRINT: 1:65536;2:0;4:131072;5:16384|15663105|0|m,a,s,p" http://localhost:%s`, port),
			Category:    "Advanced TLS",
		},
		Parameter{
			Name:        "X-USER-AGENT",
			Required:    false,
			Description: "Custom user agent string override",
			Example:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/user-agent" -H "X-USER-AGENT: Mozilla/5.0 (custom)" http://localhost:%s`, port),
			Category:    "Advanced TLS",
		},
		
		// Connection control
		Parameter{
			Name:        "X-HEADER-ORDER",
			Required:    false,
			Description: "Custom header ordering for fingerprinting",
			Example:     "accept,user-agent,accept-encoding,accept-language",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/headers" -H "X-HEADER-ORDER: accept,user-agent,accept-encoding" http://localhost:%s`, port),
			Category:    "Connection",
		},
		Parameter{
			Name:        "X-INSECURE",
			Required:    false,
			Description: "Skip TLS certificate verification",
			Example:     "true",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://self-signed.badssl.com" -H "X-INSECURE: true" http://localhost:%s`, port),
			Category:    "Connection",
		},
		Parameter{
			Name:        "X-FORCE-HTTP1",
			Required:    false,
			Description: "Force HTTP/1.1 protocol usage",
			Example:     "true",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-FORCE-HTTP1: true" http://localhost:%s`, port),
			Category:    "Connection",
		},
		Parameter{
			Name:        "X-FORCE-HTTP3",
			Required:    false,
			Description: "Force HTTP/3/QUIC protocol usage",
			Example:     "true",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-FORCE-HTTP3: true" http://localhost:%s`, port),
			Category:    "Connection",
		},
		Parameter{
			Name:        "X-ENABLE-CONNECTION-REUSE",
			Required:    false,
			Description: "Enable TCP connection reuse for performance",
			Example:     "true",
			CurlExample: fmt.Sprintf(`curl -H "X-URL: https://httpbin.org/ip" -H "X-ENABLE-CONNECTION-REUSE: true" http://localhost:%s`, port),
			Category:    "Connection",
		},
	}

	// Create list with custom delegate for better formatting
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = styles.SelectedItemStyle
	delegate.Styles.NormalTitle = styles.UnselectedItemStyle
	delegate.SetHeight(1)
	delegate.SetSpacing(0)

	paramList := list.New(parameters, delegate, 0, 0)
	paramList.SetShowHelp(false)
	paramList.SetShowStatusBar(false)
	paramList.SetFilteringEnabled(true)
	paramList.Title = "📋 Parameters"

	return ParamBrowserModel{
		list: paramList,
		port: port,
	}
}

// Init initializes the parameter browser model
func (m ParamBrowserModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the parameter browser model
func (m ParamBrowserModel) Update(msg tea.Msg) (ParamBrowserModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Calculate panel dimensions
		leftWidth := msg.Width/2 - 3
		rightWidth := msg.Width/2 - 3
		panelHeight := msg.Height - 6
		
		// Update list dimensions
		m.list.SetSize(leftWidth, panelHeight)
		
		// Initialize viewport if not ready
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
		switch msg.String() {
		case "tab":
			// Toggle focus between list and viewport
			if m.focused == 0 {
				m.focused = 1
			} else {
				m.focused = 0
			}
			return m, nil
		case "enter":
			// Update viewport content when item is selected
			m.viewport.SetContent(m.getDetailContent())
			return m, nil
		}

		// Update focused component
		if m.focused == 0 {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
			
			// Update viewport content when selection changes
			if msg.String() == "up" || msg.String() == "down" || msg.String() == "j" || msg.String() == "k" {
				m.viewport.SetContent(m.getDetailContent())
			}
		} else {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the parameter browser
func (m ParamBrowserModel) View() string {
	if !m.ready || m.width == 0 {
		return "Initializing parameter browser..."
	}

	// Get panel styles
	leftStyle := styles.GetLeftPanelStyle(m.width, m.height, m.focused == 0)
	rightStyle := styles.GetRightPanelStyle(m.width, m.height, m.focused == 1)
	
	// Render panels
	leftPanel := leftStyle.Render(m.list.View())
	rightPanel := rightStyle.Render(m.renderDetailView())
	
	// Join panels horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	
	// Add header and footer
	header := m.renderHeader()
	footer := m.renderFooter()
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the browser header
func (m ParamBrowserModel) renderHeader() string {
	style := styles.HeaderStyle.Copy().
		Width(m.width).
		Align(lipgloss.Center).
		MarginBottom(1)
	
	return style.Render("🔧 Parameter Browser - Configure your CycleTLS requests")
}

// renderFooter renders the browser footer
func (m ParamBrowserModel) renderFooter() string {
	var keys []string
	
	if m.focused == 0 {
		keys = append(keys, 
			styles.KeyStyle("↑/↓")+"Navigate",
			styles.KeyStyle("tab")+"Switch panels",
			styles.KeyStyle("/")+"Filter",
		)
	} else {
		keys = append(keys, 
			styles.KeyStyle("↑/↓")+"Scroll",
			styles.KeyStyle("tab")+"Switch panels",
		)
	}
	
	keys = append(keys, styles.KeyStyle("esc")+"Back to menu")
	
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

	param, ok := selectedItem.(Parameter)
	if !ok {
		return "Error loading parameter details"
	}

	var content strings.Builder
	
	// Parameter name with status
	nameStyle := styles.HeaderStyle.Copy()
	if param.Required {
		nameStyle = nameStyle.Foreground(styles.AccentRed)
		content.WriteString(nameStyle.Render(fmt.Sprintf("%s (REQUIRED)", param.Name)))
	} else {
		nameStyle = nameStyle.Foreground(styles.AccentGreen)
		content.WriteString(nameStyle.Render(fmt.Sprintf("%s (optional)", param.Name)))
	}
	content.WriteString("\n\n")
	
	// Category
	categoryStyle := styles.InfoStyle.Copy().MarginBottom(1)
	content.WriteString(categoryStyle.Render(fmt.Sprintf("📂 Category: %s", param.Category)))
	content.WriteString("\n\n")
	
	// Description
	descStyle := styles.ContentStyle.Copy()
	content.WriteString(descStyle.Render(param.Description))
	content.WriteString("\n\n")
	
	// Example value
	if param.Example != "" {
		exampleHeaderStyle := styles.WarningStyle.Copy()
		content.WriteString(exampleHeaderStyle.Render("💡 Example Value:"))
		content.WriteString("\n")
		
		exampleStyle := styles.CodeStyle.Copy().MarginLeft(0)
		content.WriteString(exampleStyle.Render(param.Example))
		content.WriteString("\n\n")
	}
	
	// Curl example
	if param.CurlExample != "" {
		curlHeaderStyle := styles.SuccessStyle.Copy()
		content.WriteString(curlHeaderStyle.Render("🚀 Complete curl Example:"))
		content.WriteString("\n")
		
		// Split long curl commands for better readability
		curlLines := m.formatCurlCommand(param.CurlExample)
		curlStyle := styles.CodeStyle.Copy().
			MarginLeft(0).
			Width(m.viewport.Width - 4)
		
		for _, line := range curlLines {
			content.WriteString(curlStyle.Render(line))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}
	
	// Usage notes based on parameter type
	content.WriteString(m.getUsageNotes(param))
	
	return content.String()
}

// formatCurlCommand formats long curl commands for better display
func (m ParamBrowserModel) formatCurlCommand(curlCmd string) []string {
	maxWidth := m.viewport.Width - 8
	if maxWidth < 40 {
		maxWidth = 40
	}
	
	if len(curlCmd) <= maxWidth {
		return []string{curlCmd}
	}
	
	// Split on headers to make it more readable
	parts := strings.Split(curlCmd, " -H ")
	if len(parts) == 1 {
		// No headers, just wrap normally
		return []string{curlCmd}
	}
	
	var lines []string
	lines = append(lines, parts[0]+" \\")
	
	for i := 1; i < len(parts); i++ {
		if i == len(parts)-1 {
			// Last part, no backslash
			lines = append(lines, "  -H "+parts[i])
		} else {
			lines = append(lines, "  -H "+parts[i]+" \\")
		}
	}
	
	return lines
}

// getUsageNotes provides parameter-specific usage guidance
func (m ParamBrowserModel) getUsageNotes(param Parameter) string {
	var notes strings.Builder
	
	noteStyle := styles.ContentStyle.Copy().Foreground(styles.TextMuted)
	
	switch param.Name {
	case "X-URL":
		notes.WriteString(noteStyle.Render("ℹ️  This parameter is required for all requests. Must be a valid HTTP/HTTPS URL."))
	case "X-IDENTIFIER":
		notes.WriteString(noteStyle.Render("ℹ️  Available profiles: chrome, firefox, safari_ios, safari, edge, okhttp, chrome_windows, firefox_windows, chrome_legacy_tls12"))
	case "X-SESSION-ID":
		notes.WriteString(noteStyle.Render("ℹ️  Use the same session ID across multiple requests to enable connection reuse and cookie persistence."))
	case "X-JA3":
		notes.WriteString(noteStyle.Render("ℹ️  Advanced users only. Custom JA3 fingerprint overrides the selected browser profile."))
	case "X-TIMEOUT":
		notes.WriteString(noteStyle.Render("ℹ️  Value must be between 1-300 seconds. Default is 30 seconds."))
	case "X-PROXY":
		notes.WriteString(noteStyle.Render("ℹ️  Supports HTTP, SOCKS4, SOCKS5 proxies. Format: protocol://[user:pass@]host:port"))
	default:
		notes.WriteString(noteStyle.Render("ℹ️  This parameter modifies the TLS/HTTP behavior of your requests."))
	}
	
	return notes.String()
}