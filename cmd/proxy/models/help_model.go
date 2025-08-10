// help_model.go - Help and documentation model for Bubble Tea
package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModel represents the help documentation viewer
type HelpModel struct {
	viewport viewport.Model
	ready    bool
	sections []HelpSection
}

// HelpSection represents a section in the help documentation
type HelpSection struct {
	Title   string
	Content string
}

// NewHelpModel creates a new help model with documentation content
func NewHelpModel(port string) HelpModel {
	sections := []HelpSection{
		{
			Title: "🚀 Quick Start",
			Content: fmt.Sprintf(`CycleTLS-Proxy is an HTTP proxy server that provides TLS fingerprinting capabilities.
It accepts HTTP requests with special headers to control TLS behavior and proxies 
them with specified browser fingerprints.

Basic Usage:
curl -H "X-URL: https://httpbin.org/ip" -H "X-IDENTIFIER: chrome" http://localhost:%s`, port),
		},
		{
			Title: "📋 Required Headers",
			Content: fmt.Sprintf(`X-URL: Target URL to proxy the request to (REQUIRED)
  Example: curl -H "X-URL: https://api.example.com/data" http://localhost:%s
  What it does: Specifies the destination URL for the proxy request

X-IDENTIFIER: Browser profile to use for TLS fingerprinting (OPTIONAL)
  Available: chrome, firefox, safari_ios, safari, edge, okhttp, chrome_windows, firefox_windows, chrome_legacy_tls12
  Default: chrome (if not specified)
  Example: curl -H "X-URL: https://httpbin.org/ip" -H "X-IDENTIFIER: firefox" http://localhost:%s
  What it does: Changes the TLS fingerprint to match the specified browser

X-SESSION-ID: Session identifier for connection reuse (OPTIONAL)
  Example: curl -H "X-URL: https://api.example.com" -H "X-SESSION-ID: my-session-123" http://localhost:%s
  What it does: Reuses TCP connections and maintains cookies across requests

X-PROXY: Upstream proxy to use (OPTIONAL)
  Format: http://username:password@host:port
  Example: curl -H "X-URL: https://httpbin.org/ip" -H "X-PROXY: http://user:pass@proxy.example.com:8080" http://localhost:%s
  What it does: Routes the request through an upstream proxy server

X-TIMEOUT: Custom timeout in seconds (OPTIONAL)
  Range: 1-300 seconds, Default: 30
  Example: curl -H "X-URL: https://httpbin.org/delay/10" -H "X-TIMEOUT: 15" http://localhost:%s
  What it does: Sets a custom timeout for slow requests`, port, port, port, port, port),
		},
		{
			Title: "🌐 Browser Profiles",
			Content: `Available TLS fingerprint profiles:

• chrome - Google Chrome 120 on Linux (default)
• chrome_windows - Google Chrome 120 on Windows 10/11
• firefox - Mozilla Firefox 121 on Ubuntu Linux  
• firefox_windows - Mozilla Firefox 121 on Windows 10/11
• safari - Safari 17 on macOS
• safari_ios - Safari on iOS 17.1.1
• edge - Microsoft Edge 120 on Windows 10/11
• okhttp - Android OkHttp client 4.12.0
• chrome_legacy_tls12 - Chrome with TLS 1.2 support

Each profile includes accurate JA3/JA4 fingerprints, User-Agent strings,
and HTTP/2 settings that match real browser behavior.`,
		},
		{
			Title: "🛠️ HTTP Methods & Headers",
			Content: `Supported HTTP methods:
GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS

Header Forwarding:
• All non-X-* headers are forwarded to the target URL
• Content-Type and Content-Length are preserved
• Authorization headers pass through transparently
• Custom headers are supported

Request Bodies:
• POST/PUT/PATCH requests with JSON, form data, or raw content
• Automatic content encoding/decoding (gzip, deflate, brotli)
• Streaming support for large request/response bodies`,
		},
		{
			Title: "🔧 Session Management",
			Content: `Connection Reuse:
Use X-SESSION-ID header to enable connection reuse across requests.
This improves performance and maintains consistent TLS behavior.

Example with session:
curl -H "X-URL: https://api.example.com/login" \\
     -H "X-IDENTIFIER: chrome" \\
     -H "X-SESSION-ID: my-login-session" \\
     -d '{"username":"test","password":"test"}' \\
     http://localhost:8080

curl -H "X-URL: https://api.example.com/profile" \\
     -H "X-IDENTIFIER: chrome" \\
     -H "X-SESSION-ID: my-login-session" \\
     http://localhost:8080

Session Benefits:
• TCP connection reuse
• TLS session resumption  
• Cookie persistence
• Consistent IP for multi-request flows`,
		},
		{
			Title: "📡 Advanced Examples",
			Content: `POST with JSON data:
curl -X POST http://localhost:8080 \\
  -H "X-URL: https://httpbin.org/post" \\
  -H "X-IDENTIFIER: firefox" \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Hello, World!"}'

With authentication:
curl -X GET http://localhost:8080 \\
  -H "X-URL: https://api.github.com/user" \\
  -H "X-IDENTIFIER: chrome" \\
  -H "Authorization: Bearer ghp_xxxxxxxxxxxx"

Through upstream proxy:
curl -X GET http://localhost:8080 \\
  -H "X-URL: https://httpbin.org/ip" \\
  -H "X-IDENTIFIER: safari" \\
  -H "X-PROXY: http://user:pass@proxy:8080"

Custom timeout (1-300 seconds):
curl -X GET http://localhost:8080 \\
  -H "X-URL: https://httpbin.org/delay/10" \\
  -H "X-IDENTIFIER: chrome" \\
  -H "X-TIMEOUT: 15"`,
		},
		{
			Title: "🐍 Python Example",
			Content: `import requests

def make_request(url, identifier="chrome", proxy=None, session_id=None):
    headers = {
        "X-URL": url,
        "X-IDENTIFIER": identifier
    }
    
    if proxy:
        headers["X-PROXY"] = proxy
    if session_id:
        headers["X-SESSION-ID"] = session_id
    
    response = requests.post("http://localhost:8080", headers=headers)
    return response

# Example usage
resp = make_request("https://httpbin.org/ip", identifier="firefox")
print(resp.json())`,
		},
		{
			Title: "🟨 Node.js Example", 
			Content: `const axios = require('axios');

async function makeRequest(url, options = {}) {
    const headers = {
        'X-URL': url,
        'X-IDENTIFIER': options.identifier || 'chrome'
    };
    
    if (options.proxy) headers['X-PROXY'] = options.proxy;
    if (options.sessionId) headers['X-SESSION-ID'] = options.sessionId;
    
    const response = await axios({
        method: 'POST',
        url: 'http://localhost:8080',
        headers: headers,
        data: options.data
    });
    
    return response.data;
}

// Example usage
makeRequest('https://httpbin.org/ip', {
    identifier: 'safari',
    sessionId: 'my-session'
}).then(console.log);`,
		},
		{
			Title: "🛡️ Security & Best Practices",
			Content: `Security Considerations:
• X-* configuration headers are stripped before forwarding
• Supports HTTPS targets with full certificate validation
• Session isolation prevents cross-session data leaks
• Request/response streaming prevents memory exhaustion

Performance Tips:
• Use X-SESSION-ID for multi-request workflows
• Choose appropriate browser profiles for your target sites
• Configure timeouts based on expected response times
• Monitor session count to prevent resource exhaustion

Troubleshooting:
• Check target URL is valid and accessible
• Verify browser profile identifier is correct
• Ensure upstream proxy (if used) is working
• Use /health endpoint to check proxy status`,
		},
		{
			Title: "📊 Monitoring & Health",
			Content: `Health Check Endpoint:
curl http://localhost:8080/health

Returns JSON with:
• Server status and uptime
• Available profiles count
• Active sessions count
• Configuration details

Logging:
Server logs include:
• Request details (method, target URL, profile)
• Session management (creation, reuse, cleanup)
• Error details for troubleshooting
• Performance metrics

Environment Variables:
PORT - Server listening port (default: 8080)
LOG_LEVEL - Logging level (debug, info, warn, error)`,
		},
	}

	return HelpModel{
		sections: sections,
	}
}

// Init initializes the help model
func (m HelpModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the help model
func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 2
		footerHeight := 1
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.generateContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the help model
func (m HelpModel) View() string {
	if !m.ready {
		// Initialize with default content if not ready
		return m.generateContent()
	}
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true).
		Padding(0, 1)
	
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F")).
		Padding(0, 1)
	
	header := headerStyle.Render("📚 Interactive Documentation")
	footer := footerStyle.Render("↑/↓ to scroll • ESC to return to menu")
	
	return header + "\n" + m.viewport.View() + "\n" + footer
}

// generateContent generates the formatted help content
func (m HelpModel) generateContent() string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)
	
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		MarginBottom(2).
		PaddingLeft(2)
	
	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Background(lipgloss.Color("#2D3748")).
		Padding(1).
		MarginTop(1).
		MarginBottom(1)
	
	for _, section := range m.sections {
		content.WriteString(titleStyle.Render(section.Title))
		content.WriteString("\n")
		
		// Process content to highlight code blocks
		lines := strings.Split(section.Content, "\n")
		var processedContent strings.Builder
		inCodeBlock := false
		var codeBlock strings.Builder
		
		for _, line := range lines {
			if strings.HasPrefix(line, "curl ") || strings.HasPrefix(line, "  curl ") {
				if inCodeBlock {
					processedContent.WriteString(codeStyle.Render(codeBlock.String()))
					processedContent.WriteString("\n")
					codeBlock.Reset()
				}
				codeBlock.WriteString(line + "\n")
				inCodeBlock = true
			} else if inCodeBlock && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, " ") || line == "") {
				codeBlock.WriteString(line + "\n")
			} else {
				if inCodeBlock {
					processedContent.WriteString(codeStyle.Render(codeBlock.String()))
					processedContent.WriteString("\n")
					codeBlock.Reset()
					inCodeBlock = false
				}
				processedContent.WriteString(line + "\n")
			}
		}
		
		// Handle any remaining code block
		if inCodeBlock {
			processedContent.WriteString(codeStyle.Render(codeBlock.String()))
			processedContent.WriteString("\n")
		}
		
		content.WriteString(contentStyle.Render(processedContent.String()))
		content.WriteString("\n")
	}
	
	return content.String()
}