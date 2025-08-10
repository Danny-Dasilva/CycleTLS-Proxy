// gradient_logo.go - Gradient ASCII logo renderer for CycleTLS-Proxy
package models

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BreakpointSize represents different terminal size categories
type BreakpointSize int

const (
	SizeTiny   BreakpointSize = iota // < 60 chars
	SizeSmall                        // 60-79 chars
	SizeMedium                       // 80-119 chars
	SizeLarge                        // 120-159 chars
	SizeXLarge                       // 160+ chars
)

// GradientLogo renders the CycleTLS logo with gradient effects
type GradientLogo struct {
	width      int
	height     int
	breakpoint BreakpointSize
}

// NewGradientLogo creates a new gradient logo renderer
func NewGradientLogo(width, height int) *GradientLogo {
	breakpoint := determineBreakpoint(width)
	return &GradientLogo{
		width:      width,
		height:     height,
		breakpoint: breakpoint,
	}
}

// determineBreakpoint calculates the appropriate breakpoint based on terminal width
func determineBreakpoint(width int) BreakpointSize {
	switch {
	case width < 60:
		return SizeTiny
	case width < 80:
		return SizeSmall
	case width < 120:
		return SizeMedium
	case width < 160:
		return SizeLarge
	default:
		return SizeXLarge
	}
}

// Render returns the gradient logo as a string with responsive sizing
func (g *GradientLogo) Render() string {
	switch g.breakpoint {
	case SizeTiny:
		return g.RenderTiny()
	case SizeSmall:
		return g.RenderCompact()
	case SizeMedium:
		return g.RenderStandard()
	case SizeLarge:
		return g.RenderLarge()
	case SizeXLarge:
		return g.RenderXLarge()
	default:
		return g.RenderStandard()
	}
}

// RenderTiny returns an ultra-compact logo for very small terminals
func (g *GradientLogo) RenderTiny() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#06B6D4")).
		Bold(true).
		Align(lipgloss.Center).
		Width(g.width)

	return style.Render("◯ CycleTLS ◯")
}

// RenderStandard returns the standard logo with character-level gradients
func (g *GradientLogo) RenderStandard() string {
	logo := g.getASCIILogo()
	return g.applyCharacterGradient(logo)
}

// RenderLarge returns an enhanced logo for large terminals
func (g *GradientLogo) RenderLarge() string {
	logo := g.getLargeASCIILogo()
	return g.applyCharacterGradient(logo)
}

// RenderXLarge returns the most detailed logo for extra-large terminals
func (g *GradientLogo) RenderXLarge() string {
	logo := g.getXLargeASCIILogo()
	return g.applyCharacterGradient(logo)
}

// RenderCompact returns a compact version for small terminals
func (g *GradientLogo) RenderCompact() string {
	// Compact ASCII version with character gradients
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

	return g.applyCharacterGradient(compactLogo)
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

// getLargeASCIILogo returns an enhanced logo for large terminals
func (g *GradientLogo) getLargeASCIILogo() string {
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
             ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   
                                             
          ░░▒▓ Advanced TLS Fingerprint Proxy Server ▓▒░░ `
}

// getXLargeASCIILogo returns the most detailed logo for extra-large terminals
func (g *GradientLogo) getXLargeASCIILogo() string {
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
               ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   
                                                       
     ░░▒▒▓▓ High-Performance TLS Fingerprint Proxy Server ▓▓▒▒░░ `
}

// applyCharacterGradient applies character-level gradient colors to the logo
func (g *GradientLogo) applyCharacterGradient(logo string) string {
	gradientColors := []lipgloss.Color{
		lipgloss.Color("#1E40AF"), // Blue-800
		lipgloss.Color("#1D4ED8"), // Blue-700
		lipgloss.Color("#2563EB"), // Blue-600
		lipgloss.Color("#3B82F6"), // Blue-500
		lipgloss.Color("#60A5FA"), // Blue-400
		lipgloss.Color("#06B6D4"), // Cyan-500
		lipgloss.Color("#22D3EE"), // Cyan-400
		lipgloss.Color("#67E8F9"), // Cyan-300
	}

	lines := strings.Split(logo, "\n")
	var gradientLines []string

	// Calculate total visible characters for gradient distribution
	totalChars := 0
	for _, line := range lines {
		for _, char := range line {
			if char != ' ' && char != '\t' {
				totalChars++
			}
		}
	}

	charIndex := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			gradientLines = append(gradientLines, line)
			continue
		}

		var styledLine strings.Builder
		for _, char := range line {
			if char == ' ' || char == '\t' {
				// Preserve whitespace
				styledLine.WriteRune(char)
			} else {
				// Apply gradient color based on character position
				colorIndex := (charIndex * len(gradientColors)) / totalChars
				if colorIndex >= len(gradientColors) {
					colorIndex = len(gradientColors) - 1
				}

				style := lipgloss.NewStyle().
					Foreground(gradientColors[colorIndex]).
					Bold(true)

				styledLine.WriteString(style.Render(string(char)))
				charIndex++
			}
		}

		gradientLines = append(gradientLines, styledLine.String())
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

// applyGradientToLogo applies line-level gradient colors (legacy compatibility)
func (g *GradientLogo) applyGradientToLogo(logo string) string {
	// Use character-level gradient for better consistency
	return g.applyCharacterGradient(logo)
}

// RenderWithSubtitle adds a subtitle below the logo
func (g *GradientLogo) RenderWithSubtitle(subtitle string) string {
	logo := g.Render()

	// subtitleStyle := lipgloss.NewStyle().
	// 	Foreground(lipgloss.Color("#9CA3AF")).
	// 	Align(lipgloss.Center).
	// 	Width(g.width).
	// 	MarginTop(1)

	return logo + "\n"
}
