package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/proxy"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/valyala/fasthttp"
)

var (
	// Build-time variables set by ldflags
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"

	// Color palette inspired by charmbracelet/crush
	primaryColor   = lipgloss.Color("#FF6B9D")
	secondaryColor = lipgloss.Color("#C678DD")
	accentColor    = lipgloss.Color("#61DAFB")
	successColor   = lipgloss.Color("#98D8C8")
	mutedColor     = lipgloss.Color("#6C7B7F")
)

func main() {
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("CycleTLS-Proxy %s\n", version)
		fmt.Printf("Build time: %s\n", buildTime)
		fmt.Printf("Git commit: %s\n", gitCommit)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	// Initialize logger with beautiful styling
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Prefix:          "🚀 CycleTLS",
	})
	logger.SetLevel(log.InfoLevel)

	// Get configuration
	port := getEnv("PORT", "8080")

	// Display startup banner
	displayStartupBanner(port, logger)

	// Initialize proxy handler
	handler := proxy.NewHandler(logger)
	defer handler.Close()

	// Create server
	server := &fasthttp.Server{
		Handler:                       handler.HandleRequest,
		DisablePreParseMultipartForm: true,
		StreamRequestBody:            true,
		ReadTimeout:                  30 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  60 * time.Second,
	}

	// Setup graceful shutdown
	setupGracefulShutdown(server, handler, logger)

	// Display ready message
	displayReadyMessage(port, handler.GetAvailableProfiles())

	// Start server
	logger.Info("Starting server", "port", port)
	if err := server.ListenAndServe(":" + port); err != nil {
		logger.Fatal("Server failed to start", "error", err)
	}
}

func displayStartupBanner(port string, logger *log.Logger) {
	// Main banner style
	bannerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		MarginBottom(1)

	// Create ASCII art banner
	banner := bannerStyle.Render(`
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
             ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   `)

	// Subtitle style
	subtitleStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1)

	subtitle := subtitleStyle.Render("Advanced TLS Fingerprint Proxy Server")

	// Info box style
	infoBoxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		MarginBottom(1)

	// System info
	systemInfo := fmt.Sprintf(
		"%s %s • %s • Go %s",
		lipgloss.NewStyle().Foreground(mutedColor).Render("Runtime:"),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
	)

	// Configuration info
	configInfo := fmt.Sprintf(
		"%s localhost:%s\n%s %d profiles available",
		lipgloss.NewStyle().Foreground(mutedColor).Render("Listen:"),
		port,
		lipgloss.NewStyle().Foreground(mutedColor).Render("Fingerprints:"),
		len(fingerprints.GetDefaultProfiles()),
	)

	infoBox := infoBoxStyle.Render(systemInfo + "\n" + configInfo)

	// Print banner
	fmt.Println(banner)
	fmt.Println(subtitle)
	fmt.Println(infoBox)
}

func displayReadyMessage(port string, profiles []string) {
	readyStyle := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	urlStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Underline(true)

	profileStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)

	ready := readyStyle.Render("✓ Server Ready!")
	url := urlStyle.Render(fmt.Sprintf("http://localhost:%s", port))
	profileList := profileStyle.Render(fmt.Sprintf("Available profiles: %s", strings.Join(profiles, ", ")))

	fmt.Printf("%s\n%s %s\n%s\n\n", ready, "🌐", url, profileList)
}

func setupGracefulShutdown(server *fasthttp.Server, handler *proxy.Handler, logger *log.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		shutdownStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true)

		fmt.Println("\n" + shutdownStyle.Render("🛑 Graceful shutdown initiated..."))
		logger.Info("Shutting down server")

		// Close proxy handler (closes all sessions)
		handler.Close()

		// Shutdown server
		if err := server.Shutdown(); err != nil {
			logger.Error("Error during shutdown", "error", err)
		}

		logger.Info("Server stopped")
		os.Exit(0)
	}()
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}