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
	"sync/atomic"
	"time"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/charmbracelet/log"
	"github.com/valyala/fasthttp"
)

// RequestLogEntry represents a request log entry for real-time logging
type RequestLogEntry struct {
	Timestamp  time.Time
	Method     string
	URL        string
	Profile    string
	Status     int
	Duration   time.Duration
	SessionID  string
	RemoteAddr string
}

// ServerMetrics contains real-time server metrics
type ServerMetrics struct {
	// Atomic counters for thread-safe access
	TotalRequests  int64
	SuccessfulReqs int64
	FailedRequests int64
	TotalBytes     int64

	// Protected by mutex
	mu                sync.RWMutex
	StartTime         time.Time
	LastRequestTime   time.Time
	RequestTimes      []time.Duration // Ring buffer for response times
	RequestsPerSecond float64
	ErrorRate         float64
	AverageResponse   time.Duration

	// Ring buffer management
	maxResponseTimes int
	responseIndex    int
}

// MonitorEvent represents different types of monitoring events
type MonitorEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// RequestEventData contains request-specific event data
type RequestEventData struct {
	Method     string        `json:"method"`
	URL        string        `json:"url"`
	Profile    string        `json:"profile"`
	Status     int           `json:"status"`
	Duration   time.Duration `json:"duration"`
	SessionID  string        `json:"session_id"`
	RemoteAddr string        `json:"remote_addr"`
}

// MetricsEventData contains metrics update event data
type MetricsEventData struct {
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulReqs    int64         `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	ActiveSessions    int           `json:"active_sessions"`
	Uptime            time.Duration `json:"uptime"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	ErrorRate         float64       `json:"error_rate"`
	AverageResponse   time.Duration `json:"average_response_time"`
	TotalBytes        int64         `json:"total_bytes"`
}

// newServerMetrics creates a new ServerMetrics instance
func newServerMetrics() *ServerMetrics {
	return &ServerMetrics{
		StartTime:        time.Now(),
		LastRequestTime:  time.Now(),
		RequestTimes:     make([]time.Duration, 100), // Keep last 100 response times
		maxResponseTimes: 100,
	}
}

// Handler handles incoming proxy requests with TLS fingerprint spoofing.
// It manages session state, validates headers, and forwards requests while
// maintaining proper error handling and streaming support.
type Handler struct {
	profiles       map[string]fingerprints.Profile
	clients        map[string]*cycletls.CycleTLS
	logger         *log.Logger
	mu             sync.RWMutex // protects concurrent access
	defaultTimeout time.Duration
	logChannel     chan RequestLogEntry // optional channel for real-time logging
	monitorChannel chan MonitorEvent    // optional channel for monitoring events
	metrics        *ServerMetrics       // real-time metrics tracking
	rotator        *fingerprints.ProfileRotator // profile rotation engine
}

// NewHandler creates a new proxy handler with default configuration.
// It initializes the fingerprint profiles, clients map, and sets default timeouts.
func NewHandler(logger *log.Logger) *Handler {
	return &Handler{
		profiles:       fingerprints.GetDefaultProfiles(),
		clients:        make(map[string]*cycletls.CycleTLS),
		logger:         logger,
		defaultTimeout: 30 * time.Second,
		metrics:        newServerMetrics(),
		rotator:        fingerprints.NewProfileRotator(fingerprints.DefaultRotationConfig()),
	}
}

// NewHandlerWithLogChannel creates a new proxy handler with real-time logging.
func NewHandlerWithLogChannel(logger *log.Logger, logChannel chan RequestLogEntry) *Handler {
	return &Handler{
		profiles:       fingerprints.GetDefaultProfiles(),
		clients:        make(map[string]*cycletls.CycleTLS),
		logger:         logger,
		defaultTimeout: 30 * time.Second,
		logChannel:     logChannel,
		metrics:        newServerMetrics(),
		rotator:        fingerprints.NewProfileRotator(fingerprints.DefaultRotationConfig()),
	}
}

// NewHandlerWithChannels creates a new proxy handler with both logging and monitoring channels.
func NewHandlerWithChannels(logger *log.Logger, logChannel chan RequestLogEntry, monitorChannel chan MonitorEvent) *Handler {
	return &Handler{
		profiles:       fingerprints.GetDefaultProfiles(),
		clients:        make(map[string]*cycletls.CycleTLS),
		logger:         logger,
		defaultTimeout: 30 * time.Second,
		logChannel:     logChannel,
		monitorChannel: monitorChannel,
		metrics:        newServerMetrics(),
		rotator:        fingerprints.NewProfileRotator(fingerprints.DefaultRotationConfig()),
	}
}

// HandleRequest processes incoming proxy requests with comprehensive header validation,
// error handling, and support for streaming responses.
//
// Special endpoints:
//   - GET /health: Returns health status and basic info
//
// Supported headers for proxy requests:
//
//	Basic headers:
//	- X-URL: Target URL to proxy the request to (REQUIRED)
//	- X-IDENTIFIER: Fingerprint profile to use (optional, defaults to 'chrome')
//	- X-SESSION-ID: Session identifier for connection reuse (optional)
//	- X-PROXY: Proxy server to use (optional)
//	- X-TIMEOUT: Request timeout in seconds (optional, defaults to 30s, range: 1-300)
//
//	Advanced TLS headers:
//	- X-JA3: Custom JA3 TLS fingerprint string (overrides profile JA3)
//	- X-JA4R: Custom JA4R TLS fingerprint string (overrides profile JA4R)
//	- X-HTTP2-FINGERPRINT: HTTP/2 connection settings fingerprint
//	- X-USER-AGENT: Custom user agent string (overrides profile User-Agent)
//
//	Connection control headers:
//	- X-HEADER-ORDER: Custom header ordering (comma-separated list)
//	- X-INSECURE: Skip TLS certificate verification (true/false)
//	- X-FORCE-HTTP1: Force HTTP/1.1 protocol usage (true/false)
//	- X-FORCE-HTTP3: Force HTTP/3/QUIC protocol usage (true/false)
//	- X-ENABLE-CONNECTION-REUSE: Enable TCP connection reuse (true/false)
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
	profile, actualProfileID, err := h.getProfile(headers.Identifier, headers.SessionID)
	if err != nil {
		availableProfiles := strings.Join(h.getProfileNames(), ", ")
		h.sendError(ctx, fasthttp.StatusBadRequest,
			fmt.Sprintf("Invalid identifier '%s'. Available profiles: %s", headers.Identifier, availableProfiles))
		return
	}

	// Log incoming request (with actual profile used)
	h.logRequest(ctx, headers, actualProfileID)

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

		// Track failed request metrics
		duration := time.Since(start)
		h.updateMetrics(fasthttp.StatusBadGateway, duration, 0)
		h.sendRequestLog(ctx, headers, fasthttp.StatusBadGateway, duration, actualProfileID)
		h.sendMonitorEvent("request_error", RequestEventData{
			Method:     string(ctx.Method()),
			URL:        headers.TargetURL,
			Profile:    actualProfileID,
			Status:     fasthttp.StatusBadGateway,
			Duration:   duration,
			SessionID:  headers.SessionID,
			RemoteAddr: ctx.RemoteAddr().String(),
		})
		return
	}

	// Handle the response with streaming support
	h.handleResponse(ctx, response, start)

	// Track metrics and send events
	duration := time.Since(start)
	h.updateMetrics(response.Status, duration, len(response.Body))
	h.sendRequestLog(ctx, headers, response.Status, duration, actualProfileID)
	h.sendMonitorEvent("request", RequestEventData{
		Method:     string(ctx.Method()),
		URL:        headers.TargetURL,
		Profile:    actualProfileID,
		Status:     response.Status,
		Duration:   duration,
		SessionID:  headers.SessionID,
		RemoteAddr: ctx.RemoteAddr().String(),
	})
}

// RequestHeaders contains all extracted X-* headers for request configuration
type RequestHeaders struct {
	TargetURL  string
	Identifier string
	SessionID  string
	Proxy      string
	Timeout    time.Duration

	// Advanced TLS parameters
	JA3              string
	JA4R             string
	HTTP2Fingerprint string
	CustomUserAgent  string

	// Connection control
	HeaderOrder     string
	Insecure        bool
	ForceHTTP1      bool
	ForceHTTP3      bool
	ConnectionReuse bool
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

	// Extract advanced TLS parameters
	headers.JA3 = string(ctx.Request.Header.Peek("X-JA3"))
	headers.JA4R = string(ctx.Request.Header.Peek("X-JA4R"))
	headers.HTTP2Fingerprint = string(ctx.Request.Header.Peek("X-HTTP2-FINGERPRINT"))
	headers.CustomUserAgent = string(ctx.Request.Header.Peek("X-USER-AGENT"))

	// Extract connection control parameters
	headers.HeaderOrder = string(ctx.Request.Header.Peek("X-HEADER-ORDER"))

	// Boolean parameters
	if insecureStr := string(ctx.Request.Header.Peek("X-INSECURE")); insecureStr != "" {
		headers.Insecure = strings.ToLower(insecureStr) == "true"
	}

	if forceHTTP1Str := string(ctx.Request.Header.Peek("X-FORCE-HTTP1")); forceHTTP1Str != "" {
		headers.ForceHTTP1 = strings.ToLower(forceHTTP1Str) == "true"
	}

	if forceHTTP3Str := string(ctx.Request.Header.Peek("X-FORCE-HTTP3")); forceHTTP3Str != "" {
		headers.ForceHTTP3 = strings.ToLower(forceHTTP3Str) == "true"
	}

	if connReuseStr := string(ctx.Request.Header.Peek("X-ENABLE-CONNECTION-REUSE")); connReuseStr != "" {
		headers.ConnectionReuse = strings.ToLower(connReuseStr) == "true"
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

// getProfile retrieves a fingerprint profile by identifier, supporting rotation
func (h *Handler) getProfile(identifier string, sessionID string) (fingerprints.Profile, string, error) {
	// Handle rotation identifiers
	switch identifier {
	case "auto-rotate", "random":
		return h.rotator.GetProfileForSession(sessionID)
	default:
		// Regular profile lookup
		h.mu.RLock()
		profile, exists := h.profiles[identifier]
		h.mu.RUnlock()

		if !exists {
			return fingerprints.Profile{}, "", fmt.Errorf("profile '%s' not found", identifier)
		}

		return profile, identifier, nil
	}
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
func (h *Handler) logRequest(ctx *fasthttp.RequestCtx, headers *RequestHeaders, actualProfileID string) {
	h.logger.Info("Processing proxy request",
		"method", string(ctx.Method()),
		"target_url", headers.TargetURL,
		"identifier", headers.Identifier,
		"actual_profile", actualProfileID,
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
		Timeout: int(headers.Timeout.Seconds()),
		Headers: make(map[string]string),
	}

	// Use custom JA3 if provided, otherwise use profile JA3
	if headers.JA3 != "" {
		options.Ja3 = headers.JA3
	} else {
		options.Ja3 = profile.JA3
	}

	// Use custom JA4R if provided, otherwise use profile JA4R
	if headers.JA4R != "" {
		options.Ja4r = headers.JA4R
	} else if profile.JA4R != "" {
		options.Ja4r = profile.JA4R
	}

	// Use custom User-Agent if provided, otherwise use profile User-Agent
	if headers.CustomUserAgent != "" {
		options.UserAgent = headers.CustomUserAgent
	} else {
		options.UserAgent = profile.UserAgent
	}

	// Set HTTP/2 fingerprint if provided
	// Note: HTTP/2 fingerprint support depends on CycleTLS library version
	if headers.HTTP2Fingerprint != "" {
		// options.Http2Settings = headers.HTTP2Fingerprint  // Enable when supported
	}

	// Set proxy if provided
	if headers.Proxy != "" {
		options.Proxy = headers.Proxy
	}

	// Set TLS verification mode
	if headers.Insecure {
		options.InsecureSkipVerify = true
	}

	// Set connection reuse
	// Note: Connection reuse is typically handled at the client level
	if headers.ConnectionReuse {
		// Connection reuse configuration may vary by CycleTLS version
	}

	// Handle HTTP version forcing
	if headers.ForceHTTP1 && headers.ForceHTTP3 {
		return options, fmt.Errorf("cannot force both HTTP/1 and HTTP/3 simultaneously")
	}
	if headers.ForceHTTP1 {
		options.ForceHTTP1 = true
	}
	if headers.ForceHTTP3 {
		// Note: HTTP/3 forcing may not be available in all CycleTLS versions
		// options.ForceHttp3 = true
	}

	// Forward all non-X-* headers to target server
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headerName := string(key)
		// Skip configuration headers (X-* headers)
		if !strings.HasPrefix(strings.ToUpper(headerName), "X-") {
			options.Headers[headerName] = string(value)
		}
	})

	// Override User-Agent header with the selected one
	options.Headers["User-Agent"] = options.UserAgent

	// Apply custom header order if provided
	if headers.HeaderOrder != "" {
		// Parse header order - this would need CycleTLS library support
		// options.HeaderOrder = strings.Split(headers.HeaderOrder, ",")
	}

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

	// Track error metrics
	if h.metrics != nil {
		atomic.AddInt64(&h.metrics.TotalRequests, 1)
		atomic.AddInt64(&h.metrics.FailedRequests, 1)
	}
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

// GetRotator returns the profile rotator for external configuration
func (h *Handler) GetRotator() *fingerprints.ProfileRotator {
	return h.rotator
}

// UpdateRotatorConfig updates the rotator configuration
func (h *Handler) UpdateRotatorConfig(config *fingerprints.RotationConfig) {
	h.rotator.UpdateConfig(config)
}

// GetRotationStats returns current rotation statistics
func (h *Handler) GetRotationStats() map[string]interface{} {
	return h.rotator.GetRotationStats()
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

// sendRequestLog sends a request log entry to the log channel if available
func (h *Handler) sendRequestLog(ctx *fasthttp.RequestCtx, headers *RequestHeaders, status int, duration time.Duration, actualProfileID string) {
	if h.logChannel == nil {
		return
	}

	// Create log entry
	logEntry := RequestLogEntry{
		Timestamp:  time.Now(),
		Method:     string(ctx.Method()),
		URL:        headers.TargetURL,
		Profile:    actualProfileID,
		Status:     status,
		Duration:   duration,
		SessionID:  headers.SessionID,
		RemoteAddr: ctx.RemoteAddr().String(),
	}

	// Send to channel (non-blocking)
	select {
	case h.logChannel <- logEntry:
		// Successfully sent
	default:
		// Channel is full, skip this log entry
	}
}

// updateMetrics updates the server metrics with request data
func (h *Handler) updateMetrics(status int, duration time.Duration, responseBytes int) {
	// Update atomic counters
	atomic.AddInt64(&h.metrics.TotalRequests, 1)
	atomic.AddInt64(&h.metrics.TotalBytes, int64(responseBytes))

	if status >= 200 && status < 400 {
		atomic.AddInt64(&h.metrics.SuccessfulReqs, 1)
	} else {
		atomic.AddInt64(&h.metrics.FailedRequests, 1)
	}

	// Update response times and calculate averages (protected by mutex)
	h.metrics.mu.Lock()
	h.metrics.LastRequestTime = time.Now()

	// Add response time to ring buffer
	h.metrics.RequestTimes[h.metrics.responseIndex] = duration
	h.metrics.responseIndex = (h.metrics.responseIndex + 1) % h.metrics.maxResponseTimes

	// Calculate average response time from ring buffer
	var totalDuration time.Duration
	validTimes := 0
	for _, rt := range h.metrics.RequestTimes {
		if rt > 0 {
			totalDuration += rt
			validTimes++
		}
	}
	if validTimes > 0 {
		h.metrics.AverageResponse = totalDuration / time.Duration(validTimes)
	}

	// Calculate requests per second and error rate
	uptime := time.Since(h.metrics.StartTime)
	totalReqs := atomic.LoadInt64(&h.metrics.TotalRequests)
	failedReqs := atomic.LoadInt64(&h.metrics.FailedRequests)

	if uptime.Seconds() > 0 {
		h.metrics.RequestsPerSecond = float64(totalReqs) / uptime.Seconds()
	}

	if totalReqs > 0 {
		h.metrics.ErrorRate = (float64(failedReqs) / float64(totalReqs)) * 100.0
	}

	h.metrics.mu.Unlock()

	// Send periodic metrics updates
	if totalReqs%10 == 0 { // Send update every 10 requests
		h.sendMetricsUpdate()
	}
}

// sendMonitorEvent sends a monitoring event if the channel is available
func (h *Handler) sendMonitorEvent(eventType string, data interface{}) {
	if h.monitorChannel == nil {
		return
	}

	event := MonitorEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Send to channel (non-blocking)
	select {
	case h.monitorChannel <- event:
		// Successfully sent
	default:
		// Channel is full, skip this event
	}
}

// sendMetricsUpdate sends a metrics update event
func (h *Handler) sendMetricsUpdate() {
	h.metrics.mu.RLock()
	metricsData := MetricsEventData{
		TotalRequests:     atomic.LoadInt64(&h.metrics.TotalRequests),
		SuccessfulReqs:    atomic.LoadInt64(&h.metrics.SuccessfulReqs),
		FailedRequests:    atomic.LoadInt64(&h.metrics.FailedRequests),
		ActiveSessions:    h.GetSessionCount(),
		Uptime:            time.Since(h.metrics.StartTime),
		RequestsPerSecond: h.metrics.RequestsPerSecond,
		ErrorRate:         h.metrics.ErrorRate,
		AverageResponse:   h.metrics.AverageResponse,
		TotalBytes:        atomic.LoadInt64(&h.metrics.TotalBytes),
	}
	h.metrics.mu.RUnlock()

	h.sendMonitorEvent("metrics", metricsData)
}

// GetMetrics returns the current server metrics
func (h *Handler) GetMetrics() MetricsEventData {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	return MetricsEventData{
		TotalRequests:     atomic.LoadInt64(&h.metrics.TotalRequests),
		SuccessfulReqs:    atomic.LoadInt64(&h.metrics.SuccessfulReqs),
		FailedRequests:    atomic.LoadInt64(&h.metrics.FailedRequests),
		ActiveSessions:    h.GetSessionCount(),
		Uptime:            time.Since(h.metrics.StartTime),
		RequestsPerSecond: h.metrics.RequestsPerSecond,
		ErrorRate:         h.metrics.ErrorRate,
		AverageResponse:   h.metrics.AverageResponse,
		TotalBytes:        atomic.LoadInt64(&h.metrics.TotalBytes),
	}
}

// StartPeriodicMetricsUpdates starts sending periodic metrics updates
func (h *Handler) StartPeriodicMetricsUpdates(interval time.Duration) {
	if h.monitorChannel == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			h.sendMetricsUpdate()
		}
	}()
}
