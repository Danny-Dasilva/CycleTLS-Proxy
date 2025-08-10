// gradient_logo.go - Gradient ASCII logo renderer for CycleTLS-Proxy
package models

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GradientLogo renders the CycleTLS logo with gradient effects
type GradientLogo struct {
	width  int
	height int
}

// NewGradientLogo creates a new gradient logo renderer
func NewGradientLogo(width, height int) *GradientLogo {
	return &GradientLogo{
		width:  width,
		height: height,
	}
}

// Render returns the gradient logo as a string
func (g *GradientLogo) Render() string {
	// Define gradient colors from blue to cyan
	gradientColors := []lipgloss.Color{
		lipgloss.Color("#1E40AF"), // Blue-800
		lipgloss.Color("#1D4ED8"), // Blue-700
		lipgloss.Color("#2563EB"), // Blue-600
		lipgloss.Color("#3B82F6"), // Blue-500
		lipgloss.Color("#60A5FA"), // Blue-400
		lipgloss.Color("#06B6D4"), // Cyan-500
		lipgloss.Color("#22D3EE"), // Cyan-400
	}

	logo := g.getASCIILogo()
	lines := strings.Split(logo, "\n")
	
	var gradientLines []string
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			gradientLines = append(gradientLines, line)
			continue
		}
		
		// Apply gradient color based on line position
		colorIndex := (i * len(gradientColors)) / len(lines)
		if colorIndex >= len(gradientColors) {
			colorIndex = len(gradientColors) - 1
		}
		
		style := lipgloss.NewStyle().
			Foreground(gradientColors[colorIndex]).
			Bold(true)
		
		gradientLines = append(gradientLines, style.Render(line))
	}
	
	// Center the logo if width is available
	if g.width > 0 {
		centerStyle := lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(g.width)
		return centerStyle.Render(strings.Join(gradientLines, "\n"))
	}
	
	return strings.Join(gradientLines, "\n")
}

// RenderCompact returns a compact version for small terminals
func (g *GradientLogo) RenderCompact() string {
	if g.width < 80 {
		// Very compact version for tiny terminals
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).
			Bold(true).
			Align(lipgloss.Center).
			Width(g.width)
		
		return style.Render("🔄 CycleTLS Proxy 🔄")
	}
	
	// Compact ASCII version
	compactLogo := `
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

	return g.applyGradientToLogo(compactLogo)
}

// getASCIILogo returns the full ASCII logo
func (g *GradientLogo) getASCIILogo() string {
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

// applyGradientToLogo applies gradient colors to the logo
func (g *GradientLogo) applyGradientToLogo(logo string) string {
	gradientColors := []lipgloss.Color{
		lipgloss.Color("#1E40AF"), // Blue-800
		lipgloss.Color("#1D4ED8"), // Blue-700
		lipgloss.Color("#2563EB"), // Blue-600
		lipgloss.Color("#3B82F6"), // Blue-500
		lipgloss.Color("#60A5FA"), // Blue-400
		lipgloss.Color("#06B6D4"), // Cyan-500
		lipgloss.Color("#22D3EE"), // Cyan-400
	}

	lines := strings.Split(logo, "\n")
	var gradientLines []string
	
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			gradientLines = append(gradientLines, line)
			continue
		}
		
		// Apply gradient color based on line position
		colorIndex := (i * len(gradientColors)) / len(lines)
		if colorIndex >= len(gradientColors) {
			colorIndex = len(gradientColors) - 1
		}
		
		style := lipgloss.NewStyle().
			Foreground(gradientColors[colorIndex]).
			Bold(true)
		
		gradientLines = append(gradientLines, style.Render(line))
	}
	
	// Center the logo if width is available
	if g.width > 0 {
		centerStyle := lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(g.width)
		return centerStyle.Render(strings.Join(gradientLines, "\n"))
	}
	
	return strings.Join(gradientLines, "\n")
}

// RenderWithSubtitle adds a subtitle below the logo
func (g *GradientLogo) RenderWithSubtitle(subtitle string) string {
	logo := g.Render()
	
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Align(lipgloss.Center).
		Width(g.width).
		MarginTop(1)
	
	return logo + "\n" + subtitleStyle.Render(subtitle)
}