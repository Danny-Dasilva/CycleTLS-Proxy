// Package proxy provides FastHTTP request handling with TLS fingerprint spoofing.
// It extracts X-* headers for proxy configuration, validates requests, and forwards
// non-configuration headers to target servers while maintaining session state.
package proxy

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
	"github.com/charmbracelet/log"
	"github.com/valyala/fasthttp"
)

// Handler handles incoming proxy requests with TLS fingerprint spoofing.
// It manages session state, validates headers, and forwards requests while
// maintaining proper error handling and streaming support.
type Handler struct {
	profiles       map[string]fingerprints.Profile
	clients        map[string]*cycletls.CycleTLS
	logger         *log.Logger
	mu             sync.RWMutex // protects concurrent access
	defaultTimeout time.Duration
}

// NewHandler creates a new proxy handler with default configuration.
// It initializes the fingerprint profiles, clients map, and sets default timeouts.
func NewHandler(logger *log.Logger) *Handler {
	return &Handler{
		profiles:       fingerprints.GetDefaultProfiles(),
		clients:        make(map[string]*cycletls.CycleTLS),
		logger:         logger,
		defaultTimeout: 30 * time.Second,
	}
}

// HandleRequest processes incoming proxy requests with comprehensive header validation,
// error handling, and support for streaming responses.
// 
// Special endpoints:
//   - GET /health: Returns health status and basic info
//
// Required headers for proxy requests:
//   - X-URL: Target URL to proxy the request to
//   - X-IDENTIFIER: Fingerprint profile to use (optional, defaults to 'chrome')
//   - X-SESSION-ID: Session identifier for connection reuse (optional)
//   - X-PROXY: Proxy server to use (optional)
//   - X-TIMEOUT: Request timeout in seconds (optional, defaults to 30s)
func (h *Handler) HandleRequest(ctx *fasthttp.RequestCtx) {
	// Handle health check endpoint
	if string(ctx.Path()) == "/health" {
		h.handleHealthCheck(ctx)
		return
	}

	start := time.Now()
	
	// Extract and validate X-* headers
	headers, err := h.extractHeaders(ctx)
	if err != nil {
		h.sendError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Header validation failed: %v", err))
		return
	}

	// Validate target URL
	if err := h.validateURL(headers.TargetURL); err != nil {
		h.sendError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Invalid target URL: %v", err))
		return
	}

	// Get fingerprint profile
	profile, err := h.getProfile(headers.Identifier)
	if err != nil {
		availableProfiles := strings.Join(h.getProfileNames(), ", ")
		h.sendError(ctx, fasthttp.StatusBadRequest, 
			fmt.Sprintf("Invalid identifier '%s'. Available profiles: %s", headers.Identifier, availableProfiles))
		return
	}

	// Log incoming request
	h.logRequest(ctx, headers)

	// Get or create client for this session
	client := h.getClient(headers.SessionID)

	// Build request options from profile and headers
	options, err := h.buildRequestOptions(ctx, profile, headers)
	if err != nil {
		h.sendError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Failed to build request options: %v", err))
		return
	}

	// Execute the request
	response, err := client.Do(headers.TargetURL, options, string(ctx.Method()))
	if err != nil {
		h.sendError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("Request failed: %v", err))
		h.logger.Error("Upstream request failed", 
			"error", err, 
			"target_url", headers.TargetURL,
			"method", string(ctx.Method()),
			"session_id", headers.SessionID)
		return
	}

	// Handle the response with streaming support
	h.handleResponse(ctx, response, start)
}

// RequestHeaders contains all extracted X-* headers for request configuration
type RequestHeaders struct {
	TargetURL   string
	Identifier  string
	SessionID   string
	Proxy       string
	Timeout     time.Duration
}

// extractHeaders extracts and validates all X-* configuration headers from the request
func (h *Handler) extractHeaders(ctx *fasthttp.RequestCtx) (*RequestHeaders, error) {
	headers := &RequestHeaders{}
	
	// Extract X-URL (required)
	headers.TargetURL = string(ctx.Request.Header.Peek("X-URL"))
	if headers.TargetURL == "" {
		return nil, fmt.Errorf("X-URL header is required")
	}
	
	// Extract X-IDENTIFIER (optional, defaults to 'chrome')
	headers.Identifier = string(ctx.Request.Header.Peek("X-IDENTIFIER"))
	if headers.Identifier == "" {
		headers.Identifier = "chrome"
	}
	
	// Extract X-SESSION-ID (optional)
	headers.SessionID = string(ctx.Request.Header.Peek("X-SESSION-ID"))
	
	// Extract X-PROXY (optional)
	headers.Proxy = string(ctx.Request.Header.Peek("X-PROXY"))
	
	// Extract X-TIMEOUT (optional, defaults to handler default)
	timeoutStr := string(ctx.Request.Header.Peek("X-TIMEOUT"))
	if timeoutStr == "" {
		headers.Timeout = h.defaultTimeout
	} else {
		timeoutSeconds, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid X-TIMEOUT value '%s': must be integer seconds", timeoutStr)
		}
		if timeoutSeconds < 1 || timeoutSeconds > 300 {
			return nil, fmt.Errorf("X-TIMEOUT must be between 1 and 300 seconds, got %d", timeoutSeconds)
		}
		headers.Timeout = time.Duration(timeoutSeconds) * time.Second
	}
	
	return headers, nil
}

// validateURL validates that the target URL is properly formatted and uses allowed schemes
func (h *Handler) validateURL(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %v", err)
	}
	
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme '%s': only http and https are allowed", u.Scheme)
	}
	
	if u.Host == "" {
		return fmt.Errorf("missing host in URL")
	}
	
	return nil
}

// getProfile retrieves a fingerprint profile by identifier
func (h *Handler) getProfile(identifier string) (fingerprints.Profile, error) {
	h.mu.RLock()
	profile, exists := h.profiles[identifier]
	h.mu.RUnlock()
	
	if !exists {
		return fingerprints.Profile{}, fmt.Errorf("profile '%s' not found", identifier)
	}
	
	return profile, nil
}

// getClient returns a CycleTLS client for the given session ID, creating one if needed.
// If sessionID is empty, creates a new client each time for one-time use.
func (h *Handler) getClient(sessionID string) *cycletls.CycleTLS {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Return new client for empty session ID (one-time use)
	if sessionID == "" {
		client := cycletls.Init()
		return &client
	}

	// Return existing client if found
	if client, exists := h.clients[sessionID]; exists {
		h.logger.Debug("Reusing existing session", "session_id", sessionID)
		return client
	}

	// Create new client for this session
	client := cycletls.Init()
	h.clients[sessionID] = &client
	
	h.logger.Debug("Created new session", 
		"session_id", sessionID, 
		"total_sessions", len(h.clients),
	)
	
	return &client
}

// logRequest logs the incoming request with relevant details
func (h *Handler) logRequest(ctx *fasthttp.RequestCtx, headers *RequestHeaders) {
	h.logger.Info("Processing proxy request",
		"method", string(ctx.Method()),
		"target_url", headers.TargetURL,
		"identifier", headers.Identifier,
		"session_id", headers.SessionID,
		"has_proxy", headers.Proxy != "",
		"timeout", headers.Timeout,
		"body_size", len(ctx.Request.Body()),
		"remote_addr", ctx.RemoteAddr(),
	)
}

// buildRequestOptions constructs CycleTLS options from profile and request data
func (h *Handler) buildRequestOptions(ctx *fasthttp.RequestCtx, profile fingerprints.Profile, headers *RequestHeaders) (cycletls.Options, error) {
	options := cycletls.Options{
		Ja3:         profile.JA3,
		// Ja4:         profile.JA4, // TODO: Enable when CycleTLS library supports JA4
		UserAgent:   profile.UserAgent,
		Timeout:     int(headers.Timeout.Seconds()),
		Headers:     make(map[string]string),
	}
	
	// Set proxy if provided
	if headers.Proxy != "" {
		options.Proxy = headers.Proxy
	}
	
	// Forward all non-X-* headers to target server
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headerName := string(key)
		// Skip configuration headers (X-* headers)
		if !strings.HasPrefix(strings.ToUpper(headerName), "X-") {
			options.Headers[headerName] = string(value)
		}
	})
	
	// Ensure User-Agent from profile is used (override any provided)
	options.Headers["User-Agent"] = profile.UserAgent
	
	// Set request body for methods that support it
	method := string(ctx.Method())
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if len(ctx.Request.Body()) > 0 {
			options.Body = string(ctx.Request.Body())
		}
	}
	
	return options, nil
}

// handleResponse processes the upstream response and streams it back to the client
func (h *Handler) handleResponse(ctx *fasthttp.RequestCtx, response cycletls.Response, startTime time.Time) {
	// Set response status code
	ctx.SetStatusCode(response.Status)

	// Set response headers, filtering out problematic ones
	for headerName, headerValue := range response.Headers {
		lowerName := strings.ToLower(headerName)
		
		// Skip headers that FastHTTP manages automatically
		if lowerName == "content-length" || 
		   lowerName == "transfer-encoding" || 
		   lowerName == "connection" ||
		   lowerName == "keep-alive" {
			continue
		}
		
		// Handle multiple header values properly
		if strings.Contains(headerValue, ",") && 
		   (lowerName == "set-cookie" || lowerName == "www-authenticate") {
			// For headers that can have multiple values, add them separately
			values := strings.Split(headerValue, ",")
			for _, value := range values {
				ctx.Response.Header.Add(headerName, strings.TrimSpace(value))
			}
		} else {
			ctx.Response.Header.Set(headerName, headerValue)
		}
	}

	// Stream response body for large content
	body := response.Body
	if len(body) > 0 {
		// For large responses, we could implement chunked streaming here
		// For now, set the entire body at once
		ctx.SetBodyString(body)
	}
	
	// Calculate and log response metrics
	duration := time.Since(startTime)
	h.logger.Debug("Response completed",
		"status", response.Status,
		"content_length", len(body),
		"duration_ms", duration.Milliseconds(),
		"headers_count", len(response.Headers),
	)
	
	// Log performance warning for slow requests
	if duration > 10*time.Second {
		h.logger.Warn("Slow request detected",
			"duration_ms", duration.Milliseconds(),
			"target_url", ctx.Request.Header.Peek("X-URL"),
		)
	}
}

// sendError sends a standardized error response to the client
func (h *Handler) sendError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	ctx.Error(message, statusCode)
	h.logger.Warn("Request error",
		"status_code", statusCode,
		"error", message,
		"remote_addr", ctx.RemoteAddr(),
		"method", string(ctx.Method()),
		"path", string(ctx.RequestURI()),
	)
}

// getProfileNames returns a sorted list of available profile identifiers
func (h *Handler) getProfileNames() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	names := make([]string, 0, len(h.profiles))
	for name := range h.profiles {
		names = append(names, name)
	}
	return names
}


// GetAvailableProfiles returns the list of available profile names for external use
func (h *Handler) GetAvailableProfiles() []string {
	return h.getProfileNames()
}

// GetSessionCount returns the number of active sessions
func (h *Handler) GetSessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetSessionIDs returns all active session identifiers
func (h *Handler) GetSessionIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	ids := make([]string, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// SetDefaultTimeout sets the default timeout for requests
func (h *Handler) SetDefaultTimeout(timeout time.Duration) {
	h.mu.Lock()
	h.defaultTimeout = timeout
	h.mu.Unlock()
}

// handleHealthCheck returns health status and basic proxy information
func (h *Handler) handleHealthCheck(ctx *fasthttp.RequestCtx) {
	h.mu.RLock()
	profileCount := len(h.profiles)
	defaultTimeout := h.defaultTimeout
	h.mu.RUnlock()

	activeSessionsCount := h.GetSessionCount()

	healthData := fmt.Sprintf(`{
  "status": "healthy",
  "timestamp": "%s",
  "version": "dev",
  "proxy": {
    "profiles_available": %d,
    "active_sessions": %d,
    "default_timeout": "%s"
  },
  "system": {
    "uptime": "running"
  }
}`, time.Now().UTC().Format(time.RFC3339), profileCount, activeSessionsCount, defaultTimeout)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(healthData)

	h.logger.Debug("Health check requested",
		"remote_addr", ctx.RemoteAddr(),
		"profiles", profileCount,
		"active_sessions", activeSessionsCount,
	)
}

// Close gracefully shuts down the handler and closes all active sessions
func (h *Handler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	sessionCount := len(h.clients)
	h.logger.Info("Shutting down proxy handler", "active_sessions", sessionCount)
	
	// Close all active sessions
	for sessionID, client := range h.clients {
		client.Close()
		delete(h.clients, sessionID)
	}
	
	h.logger.Info("All sessions closed successfully")
}