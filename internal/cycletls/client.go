// Package cycletls provides a CycleTLS client wrapper with enhanced session management,
// timeout handling, and proper integration with TLS fingerprint profiles.
// It manages connection reuse, handles cleanup, and provides a high-level interface
// for making requests with specific TLS fingerprints.
package cycletls

import (
	"fmt"
	"sync"
	"time"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/charmbracelet/log"
)

// Client wraps the CycleTLS client with enhanced functionality including
// session management, timeout handling, and connection reuse capabilities.
type Client struct {
	underlying   *cycletls.CycleTLS
	sessionID    string
	createdAt    time.Time
	lastUsedAt   time.Time
	requestCount int
	mu           sync.RWMutex // protects concurrent access to client state
	closed       bool
	logger       *log.Logger
}

// NewClient creates a new CycleTLS client wrapper with session management capabilities.
// If sessionID is empty, creates a temporary client for one-time use.
func NewClient(sessionID string) *Client {
	client := cycletls.Init()
	now := time.Now()

	return &Client{
		underlying:   &client,
		sessionID:    sessionID,
		createdAt:    now,
		lastUsedAt:   now,
		requestCount: 0,
		closed:       false,
		logger:       log.New(nil), // Default logger, can be set later
	}
}

// NewClientWithLogger creates a new client with a specific logger
func NewClientWithLogger(sessionID string, logger *log.Logger) *Client {
	client := NewClient(sessionID)
	client.logger = logger
	return client
}

// Do performs an HTTP request using the CycleTLS client with proper error handling
// and session state management. It tracks usage statistics and handles timeouts.
func (c *Client) Do(url string, options cycletls.Options, method string) (cycletls.Response, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return cycletls.Response{}, fmt.Errorf("client is closed")
	}
	c.lastUsedAt = time.Now()
	c.requestCount++
	requestNum := c.requestCount
	c.mu.Unlock()

	// Set default timeout if not specified
	if options.Timeout == 0 {
		options.Timeout = 30 // Default 30 seconds
	}

	// Log request for debugging
	c.logger.Debug("Making request",
		"session_id", c.sessionID,
		"request_num", requestNum,
		"method", method,
		"url", url,
		"timeout", options.Timeout,
		"has_proxy", options.Proxy != "",
	)

	start := time.Now()
	response, err := c.underlying.Do(url, options, method)
	duration := time.Since(start)

	if err != nil {
		c.logger.Debug("Request failed",
			"session_id", c.sessionID,
			"request_num", requestNum,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return response, fmt.Errorf("request failed: %w", err)
	}

	c.logger.Debug("Request completed",
		"session_id", c.sessionID,
		"request_num", requestNum,
		"status", response.Status,
		"duration_ms", duration.Milliseconds(),
		"response_size", len(response.Body),
	)

	return response, nil
}

// Convenience methods for common HTTP methods with proper error handling

// Get performs a GET request with the configured TLS fingerprint
func (c *Client) Get(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "GET")
}

// Post performs a POST request with the configured TLS fingerprint
func (c *Client) Post(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "POST")
}

// Put performs a PUT request with the configured TLS fingerprint
func (c *Client) Put(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "PUT")
}

// Delete performs a DELETE request with the configured TLS fingerprint
func (c *Client) Delete(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "DELETE")
}

// Patch performs a PATCH request with the configured TLS fingerprint
func (c *Client) Patch(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "PATCH")
}

// Head performs a HEAD request with the configured TLS fingerprint
func (c *Client) Head(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "HEAD")
}

// Options performs an OPTIONS request with the configured TLS fingerprint
func (c *Client) Options(url string, options cycletls.Options) (cycletls.Response, error) {
	return c.Do(url, options, "OPTIONS")
}

// GetSessionID returns the session ID for this client
func (c *Client) GetSessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// GetCreatedAt returns when this client was created
func (c *Client) GetCreatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.createdAt
}

// GetLastUsedAt returns when this client was last used
func (c *Client) GetLastUsedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUsedAt
}

// GetRequestCount returns the number of requests made with this client
func (c *Client) GetRequestCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.requestCount
}

// GetAge returns how long this client has been active
func (c *Client) GetAge() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.createdAt)
}

// IsIdle returns true if the client hasn't been used for the specified duration
func (c *Client) IsIdle(duration time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastUsedAt) > duration
}

// IsClosed returns whether this client has been closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// SetLogger sets the logger for this client
func (c *Client) SetLogger(logger *log.Logger) {
	c.mu.Lock()
	c.logger = logger
	c.mu.Unlock()
}

// Close gracefully closes the underlying CycleTLS client and marks it as closed
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil // Already closed
	}

	c.logger.Debug("Closing client",
		"session_id", c.sessionID,
		"age_seconds", time.Since(c.createdAt).Seconds(),
		"request_count", c.requestCount,
	)

	c.underlying.Close()
	c.closed = true
	return nil
}

// ClientConfig holds configuration for client creation and management
type ClientConfig struct {
	DefaultTimeout  time.Duration
	MaxIdleTime     time.Duration
	CleanupInterval time.Duration
	MaxSessionAge   time.Duration
	EnableCleanup   bool
	Logger          *log.Logger
}

// DefaultClientConfig returns a ClientConfig with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		DefaultTimeout:  30 * time.Second,
		MaxIdleTime:     300 * time.Second,  // 5 minutes
		CleanupInterval: 60 * time.Second,   // 1 minute
		MaxSessionAge:   3600 * time.Second, // 1 hour
		EnableCleanup:   true,
		Logger:          log.New(nil),
	}
}

// ClientManager manages multiple CycleTLS clients for session reuse with
// automatic cleanup, idle timeout handling, and performance monitoring.
type ClientManager struct {
	clients     map[string]*Client
	mu          sync.RWMutex
	config      *ClientConfig
	cleanupStop chan bool
	closed      bool
}

// NewClientManager creates a new client manager with default configuration
func NewClientManager() *ClientManager {
	return NewClientManagerWithConfig(DefaultClientConfig())
}

// NewClientManagerWithConfig creates a new client manager with custom configuration
func NewClientManagerWithConfig(config *ClientConfig) *ClientManager {
	cm := &ClientManager{
		clients:     make(map[string]*Client),
		config:      config,
		cleanupStop: make(chan bool),
		closed:      false,
	}

	// Start cleanup goroutine if enabled
	if config.EnableCleanup {
		go cm.cleanupRoutine()
	}

	return cm
}

// cleanupRoutine periodically cleans up idle and old sessions
func (cm *ClientManager) cleanupRoutine() {
	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.performCleanup()
		case <-cm.cleanupStop:
			return
		}
	}
}

// performCleanup removes idle and old clients
func (cm *ClientManager) performCleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		return
	}

	var toRemove []string
	now := time.Now()

	for sessionID, client := range cm.clients {
		// Remove clients that are too old or have been idle too long
		if client.GetAge() > cm.config.MaxSessionAge || client.IsIdle(cm.config.MaxIdleTime) {
			toRemove = append(toRemove, sessionID)
		}
	}

	// Clean up identified clients
	for _, sessionID := range toRemove {
		client := cm.clients[sessionID]
		client.Close()
		delete(cm.clients, sessionID)

		cm.config.Logger.Debug("Cleaned up idle session",
			"session_id", sessionID,
			"age_seconds", client.GetAge().Seconds(),
			"idle_seconds", now.Sub(client.GetLastUsedAt()).Seconds(),
			"request_count", client.GetRequestCount(),
		)
	}

	if len(toRemove) > 0 {
		cm.config.Logger.Info("Cleanup completed",
			"removed_sessions", len(toRemove),
			"active_sessions", len(cm.clients),
		)
	}
}

// GetClient returns a client for the given session ID, creating one if needed.
// If sessionID is empty, returns a temporary client for one-time use.
func (cm *ClientManager) GetClient(sessionID string) *Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		// Return a temporary client if manager is closed
		return NewClientWithLogger("", cm.config.Logger)
	}

	// Return temporary client for empty session ID
	if sessionID == "" {
		return NewClientWithLogger("", cm.config.Logger)
	}

	// Return existing client if found and not closed
	if client, exists := cm.clients[sessionID]; exists && !client.IsClosed() {
		cm.config.Logger.Debug("Reusing existing session", "session_id", sessionID)
		return client
	}

	// Create new client
	client := NewClientWithLogger(sessionID, cm.config.Logger)
	cm.clients[sessionID] = client

	cm.config.Logger.Debug("Created new session",
		"session_id", sessionID,
		"total_sessions", len(cm.clients),
	)

	return client
}

// GetClientWithProfile creates or returns a client configured with a specific fingerprint profile
func (cm *ClientManager) GetClientWithProfile(sessionID string, profile fingerprints.Profile) *Client {
	client := cm.GetClient(sessionID)
	// Note: Profile configuration is handled at the request level in options
	// This method exists for future extensibility
	return client
}

// RemoveClient removes a specific client from the manager
func (cm *ClientManager) RemoveClient(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if client, exists := cm.clients[sessionID]; exists {
		client.Close()
		delete(cm.clients, sessionID)

		cm.config.Logger.Debug("Removed session",
			"session_id", sessionID,
			"remaining_sessions", len(cm.clients),
		)
	}
}

// RemoveIdleClients removes all clients that have been idle for longer than the specified duration
func (cm *ClientManager) RemoveIdleClients(idleDuration time.Duration) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var removed []string
	for sessionID, client := range cm.clients {
		if client.IsIdle(idleDuration) {
			client.Close()
			removed = append(removed, sessionID)
		}
	}

	for _, sessionID := range removed {
		delete(cm.clients, sessionID)
	}

	if len(removed) > 0 {
		cm.config.Logger.Info("Removed idle clients",
			"count", len(removed),
			"idle_duration", idleDuration,
			"remaining_sessions", len(cm.clients),
		)
	}

	return len(removed)
}

// GetActiveSessionsCount returns the number of active sessions
func (cm *ClientManager) GetActiveSessionsCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.clients)
}

// GetSessionIDs returns all active session identifiers
func (cm *ClientManager) GetSessionIDs() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ids := make([]string, 0, len(cm.clients))
	for id := range cm.clients {
		ids = append(ids, id)
	}
	return ids
}

// SessionStats contains detailed statistics about a session
type SessionStats struct {
	SessionID    string    `json:"session_id"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	RequestCount int       `json:"request_count"`
	Age          string    `json:"age"`
	IdleTime     string    `json:"idle_time"`
}

// GetSessionStats returns statistics for a specific session
func (cm *ClientManager) GetSessionStats(sessionID string) (*SessionStats, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	client, exists := cm.clients[sessionID]
	if !exists {
		return nil, false
	}

	stats := &SessionStats{
		SessionID:    sessionID,
		CreatedAt:    client.GetCreatedAt(),
		LastUsedAt:   client.GetLastUsedAt(),
		RequestCount: client.GetRequestCount(),
		Age:          client.GetAge().String(),
		IdleTime:     time.Since(client.GetLastUsedAt()).String(),
	}

	return stats, true
}

// GetAllSessionStats returns statistics for all active sessions
func (cm *ClientManager) GetAllSessionStats() map[string]SessionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]SessionStats)
	for sessionID, client := range cm.clients {
		stats[sessionID] = SessionStats{
			SessionID:    sessionID,
			CreatedAt:    client.GetCreatedAt(),
			LastUsedAt:   client.GetLastUsedAt(),
			RequestCount: client.GetRequestCount(),
			Age:          client.GetAge().String(),
			IdleTime:     time.Since(client.GetLastUsedAt()).String(),
		}
	}

	return stats
}

// CloseAll gracefully closes all managed clients and stops cleanup routines
func (cm *ClientManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		return
	}

	// Stop cleanup routine
	if cm.config.EnableCleanup {
		close(cm.cleanupStop)
	}

	// Close all clients
	for sessionID, client := range cm.clients {
		client.Close()
		delete(cm.clients, sessionID)
	}

	cm.closed = true
	cm.config.Logger.Info("Client manager closed", "closed_sessions", len(cm.clients))
}

// IsClosed returns whether the client manager has been closed
func (cm *ClientManager) IsClosed() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.closed
}

// SetConfig updates the client manager configuration
func (cm *ClientManager) SetConfig(config *ClientConfig) {
	cm.mu.Lock()
	cm.config = config
	cm.mu.Unlock()
}

// GetConfig returns the current client manager configuration
func (cm *ClientManager) GetConfig() *ClientConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}
