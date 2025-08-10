// monitor_model.go - Live monitoring model for Bubble Tea
package models

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MonitorModel represents the live monitoring dashboard
type MonitorModel struct {
	width       int
	height      int
	lastUpdate  time.Time
	stats       ServerStats
	requests    []RequestLog
	maxRequests int
}

// ServerStats represents server statistics
type ServerStats struct {
	Uptime         time.Duration
	TotalRequests  int64
	ActiveSessions int
	RequestsPerSec float64
	ErrorRate      float64
	AvgResponseTime time.Duration
}

// RequestLog represents a logged request
type RequestLog struct {
	Timestamp   time.Time
	Method      string
	URL         string
	Profile     string
	Status      int
	Duration    time.Duration
	SessionID   string
}

// tickMsg is sent periodically to update the display
type tickMsg time.Time

// NewMonitorModel creates a new monitoring model
func NewMonitorModel() MonitorModel {
	return MonitorModel{
		lastUpdate:  time.Now(),
		maxRequests: 50,
		requests:    make([]RequestLog, 0, 50),
		stats: ServerStats{
			Uptime: 0,
		},
	}
}

// Init initializes the monitor model
func (m MonitorModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages for the monitor model
func (m MonitorModel) Update(msg tea.Msg) (MonitorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
	case tickMsg:
		// Update stats (in a real implementation, this would fetch from the server)
		m.updateStats()
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	
	return m, nil
}

// View renders the monitor model
func (m MonitorModel) View() string {
	var content strings.Builder
	
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#61DAFB")).
		Bold(true).
		Padding(0, 1)
	
	header := headerStyle.Render("📊 Live Server Monitor")
	content.WriteString(header)
	content.WriteString("\n\n")
	
	// Stats grid
	content.WriteString(m.renderStatsGrid())
	content.WriteString("\n\n")
	
	// Recent requests
	content.WriteString(m.renderRequestLog())
	
	// Footer with last update time
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F")).
		Align(lipgloss.Center).
		MarginTop(2)
	
	footer := footerStyle.Render(fmt.Sprintf("Last updated: %s • Auto-refresh every 1s • ESC to return", 
		m.lastUpdate.Format("15:04:05")))
	
	content.WriteString("\n")
	content.WriteString(footer)
	
	return content.String()
}

// renderStatsGrid renders the statistics in a grid layout
func (m MonitorModel) renderStatsGrid() string {
	// Create stat boxes
	statBoxes := []struct {
		title string
		value string
		color lipgloss.Color
	}{
		{
			title: "Uptime",
			value: formatDuration(m.stats.Uptime),
			color: lipgloss.Color("#98D8C8"),
		},
		{
			title: "Total Requests",
			value: fmt.Sprintf("%d", m.stats.TotalRequests),
			color: lipgloss.Color("#61DAFB"),
		},
		{
			title: "Active Sessions",
			value: fmt.Sprintf("%d", m.stats.ActiveSessions),
			color: lipgloss.Color("#FF6B9D"),
		},
		{
			title: "Requests/sec",
			value: fmt.Sprintf("%.1f", m.stats.RequestsPerSec),
			color: lipgloss.Color("#C678DD"),
		},
		{
			title: "Error Rate",
			value: fmt.Sprintf("%.1f%%", m.stats.ErrorRate),
			color: getErrorRateColor(m.stats.ErrorRate),
		},
		{
			title: "Avg Response",
			value: formatDuration(m.stats.AvgResponseTime),
			color: lipgloss.Color("#FFB86C"),
		},
	}
	
	// Calculate box dimensions
	boxWidth := 18
	if m.width > 120 {
		boxWidth = (m.width - 10) / 3 // 3 columns
	} else if m.width > 80 {
		boxWidth = (m.width - 8) / 2 // 2 columns  
	}
	
	var rows []string
	var currentRow []string
	
	for i, stat := range statBoxes {
		boxStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(stat.color).
			Padding(1, 2).
			Width(boxWidth).
			Align(lipgloss.Center)
		
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7B7F")).
			Align(lipgloss.Center)
		
		valueStyle := lipgloss.NewStyle().
			Foreground(stat.color).
			Bold(true).
			Align(lipgloss.Center)
		
		boxContent := titleStyle.Render(stat.title) + "\n" + valueStyle.Render(stat.value)
		box := boxStyle.Render(boxContent)
		
		currentRow = append(currentRow, box)
		
		// Create rows of 3 boxes (or 2 on smaller screens)
		maxCols := 3
		if m.width <= 120 {
			maxCols = 2
		}
		if m.width <= 80 {
			maxCols = 1
		}
		
		if len(currentRow) == maxCols || i == len(statBoxes)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = []string{}
		}
	}
	
	return strings.Join(rows, "\n")
}

// renderRequestLog renders the recent request log
func (m MonitorModel) renderRequestLog() string {
	var content strings.Builder
	
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B9D")).
		Bold(true).
		MarginBottom(1)
	
	content.WriteString(headerStyle.Render("🕐 Recent Requests"))
	content.WriteString("\n\n")
	
	if len(m.requests) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7B7F")).
			Italic(true)
		
		content.WriteString(emptyStyle.Render("No recent requests to display"))
		return content.String()
	}
	
	// Table header
	headerRow := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7B7F")).
		Bold(true).
		Render("TIME     METHOD  STATUS  PROFILE         URL")
	
	content.WriteString(headerRow)
	content.WriteString("\n")
	
	// Table rows
	for i := len(m.requests) - 1; i >= 0 && len(m.requests)-i <= 15; i-- {
		req := m.requests[i]
		
		timeStr := req.Timestamp.Format("15:04:05")
		methodStr := fmt.Sprintf("%-6s", req.Method)
		statusStr := fmt.Sprintf("%d", req.Status)
		profileStr := fmt.Sprintf("%-12s", req.Profile)
		urlStr := req.URL
		
		// Truncate URL if too long
		maxURLLen := m.width - 45
		if maxURLLen > 20 && len(urlStr) > maxURLLen {
			urlStr = urlStr[:maxURLLen-3] + "..."
		}
		
		// Color status based on HTTP status code
		var statusColor lipgloss.Color
		if req.Status >= 200 && req.Status < 300 {
			statusColor = lipgloss.Color("#98D8C8") // Green
		} else if req.Status >= 400 && req.Status < 500 {
			statusColor = lipgloss.Color("#FFB86C") // Orange
		} else if req.Status >= 500 {
			statusColor = lipgloss.Color("#FF5555") // Red
		} else {
			statusColor = lipgloss.Color("#6C7B7F") // Gray
		}
		
		row := fmt.Sprintf("%s %s %s %s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7B7F")).Render(timeStr),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#61DAFB")).Render(methodStr),
			lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusStr),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD")).Render(profileStr),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(urlStr),
		)
		
		content.WriteString(row)
		content.WriteString("\n")
	}
	
	return content.String()
}

// updateStats updates the server statistics (mock implementation)
func (m *MonitorModel) updateStats() {
	m.lastUpdate = time.Now()
	
	// Mock data - in real implementation, this would fetch from the actual server
	m.stats.Uptime += time.Second
	m.stats.TotalRequests += int64(m.generateRandomNumber(0, 3))
	m.stats.ActiveSessions = m.generateRandomNumber(0, 15)
	m.stats.RequestsPerSec = float64(m.generateRandomNumber(0, 50)) / 10.0
	m.stats.ErrorRate = float64(m.generateRandomNumber(0, 100)) / 100.0
	m.stats.AvgResponseTime = time.Duration(m.generateRandomNumber(50, 500)) * time.Millisecond
	
	// Add mock request logs occasionally
	if m.generateRandomNumber(1, 10) <= 3 { // 30% chance
		m.addMockRequest()
	}
}

// addMockRequest adds a mock request to the log
func (m *MonitorModel) addMockRequest() {
	profiles := []string{"chrome", "firefox", "safari", "edge", "okhttp"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	urls := []string{
		"https://httpbin.org/ip",
		"https://api.github.com/user",
		"https://example.com/api/data",
		"https://jsonplaceholder.typicode.com/posts/1",
	}
	statuses := []int{200, 201, 400, 404, 500}
	
	req := RequestLog{
		Timestamp: time.Now(),
		Method:    methods[m.generateRandomNumber(0, len(methods)-1)],
		URL:       urls[m.generateRandomNumber(0, len(urls)-1)],
		Profile:   profiles[m.generateRandomNumber(0, len(profiles)-1)],
		Status:    statuses[m.generateRandomNumber(0, len(statuses)-1)],
		Duration:  time.Duration(m.generateRandomNumber(50, 1000)) * time.Millisecond,
		SessionID: fmt.Sprintf("sess-%d", m.generateRandomNumber(1000, 9999)),
	}
	
	m.requests = append(m.requests, req)
	
	// Keep only the most recent requests
	if len(m.requests) > m.maxRequests {
		m.requests = m.requests[1:]
	}
}

// generateRandomNumber generates a random number (mock implementation)
func (m *MonitorModel) generateRandomNumber(min, max int) int {
	// Simple pseudo-random based on current time
	seed := time.Now().UnixNano() % 1000
	return min + int(seed)%(max-min+1)
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// getErrorRateColor returns an appropriate color for the error rate
func getErrorRateColor(rate float64) lipgloss.Color {
	if rate <= 1.0 {
		return lipgloss.Color("#98D8C8") // Green
	} else if rate <= 5.0 {
		return lipgloss.Color("#FFB86C") // Orange
	} else {
		return lipgloss.Color("#FF5555") // Red
	}
}