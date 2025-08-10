package main

/*
CycleTLS-Proxy Go Client Examples

This package provides comprehensive examples and a Go client library
for interacting with the CycleTLS-Proxy server. It demonstrates various
use cases including different browser profiles, session management,
authentication, and error handling.

Usage:
    go run examples/go.go

Or import as a library:
    import "path/to/examples"
    client := examples.NewCycleTLSClient("http://localhost:8080")
    response, err := client.Get("https://httpbin.org/json", examples.WithProfile(examples.ProfileChrome))
*/

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Browser profiles available for TLS fingerprinting
const (
	ProfileChrome          = "chrome"
	ProfileChromeWindows   = "chrome_windows"
	ProfileFirefox         = "firefox"
	ProfileFirefoxWindows  = "firefox_windows"
	ProfileSafari          = "safari"
	ProfileSafariIOS       = "safari_ios"
	ProfileEdge            = "edge"
	ProfileOkHttp          = "okhttp"
	ProfileChromeLegacyTLS = "chrome_legacy_tls12"
)

// Custom error types
var (
	ErrCycleTLS               = errors.New("cycletls error")
	ErrCycleTLSTimeout        = errors.New("cycletls timeout")
	ErrCycleTLSInvalidProfile = errors.New("cycletls invalid profile")
	ErrCycleTLSConnection     = errors.New("cycletls connection error")
)

// CycleTLSError wraps errors with additional context
type CycleTLSError struct {
	Type    error
	Message string
	Code    int
}

func (e *CycleTLSError) Error() string {
	return fmt.Sprintf("%v: %s", e.Type, e.Message)
}

func (e *CycleTLSError) Unwrap() error {
	return e.Type
}

// RequestConfig contains configuration for proxy requests
type RequestConfig struct {
	URL           string
	Profile       string
	SessionID     string
	UpstreamProxy string
	Timeout       int
	Headers       map[string]string
}

// RequestOption is a functional option for configuring requests
type RequestOption func(*RequestConfig)

// WithProfile sets the browser profile to use
func WithProfile(profile string) RequestOption {
	return func(c *RequestConfig) {
		c.Profile = profile
	}
}

// WithSessionID sets the session ID for connection reuse
func WithSessionID(sessionID string) RequestOption {
	return func(c *RequestConfig) {
		c.SessionID = sessionID
	}
}

// WithUpstreamProxy sets an upstream proxy server
func WithUpstreamProxy(proxy string) RequestOption {
	return func(c *RequestConfig) {
		c.UpstreamProxy = proxy
	}
}

// WithTimeout sets the request timeout in seconds
func WithTimeout(timeout int) RequestOption {
	return func(c *RequestConfig) {
		c.Timeout = timeout
	}
}

// WithHeaders sets additional headers to send
func WithHeaders(headers map[string]string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithHeader sets a single additional header
func WithHeader(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers[key] = value
	}
}

// CycleTLSClient is a Go client for the CycleTLS-Proxy server
type CycleTLSClient struct {
	proxyURL       string
	defaultTimeout int
	httpClient     *http.Client
	mu             sync.RWMutex
}

// NewCycleTLSClient creates a new CycleTLS client
func NewCycleTLSClient(proxyURL string) *CycleTLSClient {
	return &CycleTLSClient{
		proxyURL:       strings.TrimRight(proxyURL, "/"),
		defaultTimeout: 30,
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // Add buffer for proxy overhead
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects, let the proxy handle them
				return http.ErrUseLastResponse
			},
		},
	}
}

// SetDefaultTimeout sets the default timeout for requests
func (c *CycleTLSClient) SetDefaultTimeout(timeout int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultTimeout = timeout
	c.httpClient.Timeout = time.Duration(timeout+5) * time.Second
}

// makeHeaders creates the appropriate headers for the proxy request
func (c *CycleTLSClient) makeHeaders(config *RequestConfig) map[string]string {
	headers := map[string]string{
		"X-URL":        config.URL,
		"X-IDENTIFIER": config.Profile,
	}

	if config.SessionID != "" {
		headers["X-SESSION-ID"] = config.SessionID
	}

	if config.UpstreamProxy != "" {
		headers["X-PROXY"] = config.UpstreamProxy
	}

	if config.Timeout != 0 && config.Timeout != c.defaultTimeout {
		headers["X-TIMEOUT"] = strconv.Itoa(config.Timeout)
	}

	// Add custom headers
	for k, v := range config.Headers {
		headers[k] = v
	}

	return headers
}

// handleError processes and wraps HTTP errors appropriately
func (c *CycleTLSClient) handleError(resp *http.Response, err error) error {
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return &CycleTLSError{
				Type:    ErrCycleTLSTimeout,
				Message: "request timed out",
			}
		}
		return &CycleTLSError{
			Type:    ErrCycleTLSConnection,
			Message: err.Error(),
		}
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		message := string(body)

		switch resp.StatusCode {
		case 400:
			if strings.Contains(message, "Invalid identifier") {
				return &CycleTLSError{
					Type:    ErrCycleTLSInvalidProfile,
					Message: message,
					Code:    resp.StatusCode,
				}
			}
			if strings.Contains(strings.ToLower(message), "timeout") {
				return &CycleTLSError{
					Type:    ErrCycleTLSTimeout,
					Message: message,
					Code:    resp.StatusCode,
				}
			}
			return &CycleTLSError{
				Type:    ErrCycleTLS,
				Message: fmt.Sprintf("bad request: %s", message),
				Code:    resp.StatusCode,
			}
		case 502:
			return &CycleTLSError{
				Type:    ErrCycleTLS,
				Message: fmt.Sprintf("upstream request failed: %s", message),
				Code:    resp.StatusCode,
			}
		default:
			return &CycleTLSError{
				Type:    ErrCycleTLS,
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, message),
				Code:    resp.StatusCode,
			}
		}
	}

	return nil
}

// Request makes a request through the CycleTLS-Proxy
func (c *CycleTLSClient) Request(method, targetURL string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	// Apply default configuration
	config := &RequestConfig{
		URL:     targetURL,
		Profile: ProfileChrome,
		Timeout: c.defaultTimeout,
	}

	// Apply options
	for _, option := range options {
		option(config)
	}

	// Validate URL
	if _, err := url.Parse(config.URL); err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("invalid URL: %v", err),
		}
	}

	// Create the proxy request
	req, err := http.NewRequest(method, c.proxyURL, body)
	if err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	// Set headers
	headers := c.makeHeaders(config)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Update client timeout if needed
	if config.Timeout != c.defaultTimeout {
		client := &http.Client{
			Timeout:       time.Duration(config.Timeout+5) * time.Second,
			CheckRedirect: c.httpClient.CheckRedirect,
		}
		resp, err := client.Do(req)
		return resp, c.handleError(resp, err)
	}

	// Use default client
	resp, err := c.httpClient.Do(req)
	return resp, c.handleError(resp, err)
}

// Get makes a GET request
func (c *CycleTLSClient) Get(url string, options ...RequestOption) (*http.Response, error) {
	return c.Request("GET", url, nil, options...)
}

// Post makes a POST request with optional body
func (c *CycleTLSClient) Post(url string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	return c.Request("POST", url, body, options...)
}

// PostJSON makes a POST request with JSON body
func (c *CycleTLSClient) PostJSON(url string, data interface{}, options ...RequestOption) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("failed to marshal JSON: %v", err),
		}
	}

	options = append(options, WithHeader("Content-Type", "application/json"))
	return c.Post(url, bytes.NewReader(jsonData), options...)
}

// Put makes a PUT request
func (c *CycleTLSClient) Put(url string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	return c.Request("PUT", url, body, options...)
}

// PutJSON makes a PUT request with JSON body
func (c *CycleTLSClient) PutJSON(url string, data interface{}, options ...RequestOption) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("failed to marshal JSON: %v", err),
		}
	}

	options = append(options, WithHeader("Content-Type", "application/json"))
	return c.Put(url, bytes.NewReader(jsonData), options...)
}

// Patch makes a PATCH request
func (c *CycleTLSClient) Patch(url string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	return c.Request("PATCH", url, body, options...)
}

// PatchJSON makes a PATCH request with JSON body
func (c *CycleTLSClient) PatchJSON(url string, data interface{}, options ...RequestOption) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("failed to marshal JSON: %v", err),
		}
	}

	options = append(options, WithHeader("Content-Type", "application/json"))
	return c.Patch(url, bytes.NewReader(jsonData), options...)
}

// Delete makes a DELETE request
func (c *CycleTLSClient) Delete(url string, options ...RequestOption) (*http.Response, error) {
	return c.Request("DELETE", url, nil, options...)
}

// Head makes a HEAD request
func (c *CycleTLSClient) Head(url string, options ...RequestOption) (*http.Response, error) {
	return c.Request("HEAD", url, nil, options...)
}

// HealthCheck checks the health of the CycleTLS-Proxy server
func (c *CycleTLSClient) HealthCheck() (map[string]interface{}, error) {
	resp, err := http.Get(c.proxyURL + "/health")
	if err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLSConnection,
			Message: fmt.Sprintf("health check failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("health check returned HTTP %d", resp.StatusCode),
		}
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("failed to decode health response: %v", err),
		}
	}

	return health, nil
}

// SessionContext provides session management for persistent connections
type SessionContext struct {
	client    *CycleTLSClient
	sessionID string
}

// NewSession creates a new session context
func (c *CycleTLSClient) NewSession(sessionID string) *SessionContext {
	if sessionID == "" {
		sessionID = fmt.Sprintf("go-session-%s", uuid.New().String()[:8])
	}

	return &SessionContext{
		client:    c,
		sessionID: sessionID,
	}
}

// SessionID returns the session ID
func (s *SessionContext) SessionID() string {
	return s.sessionID
}

// Request makes a request using this session
func (s *SessionContext) Request(method, url string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	options = append(options, WithSessionID(s.sessionID))
	return s.client.Request(method, url, body, options...)
}

// Get makes a GET request using this session
func (s *SessionContext) Get(url string, options ...RequestOption) (*http.Response, error) {
	return s.Request("GET", url, nil, options...)
}

// Post makes a POST request using this session
func (s *SessionContext) Post(url string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	return s.Request("POST", url, body, options...)
}

// PostJSON makes a POST request with JSON body using this session
func (s *SessionContext) PostJSON(url string, data interface{}, options ...RequestOption) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, &CycleTLSError{
			Type:    ErrCycleTLS,
			Message: fmt.Sprintf("failed to marshal JSON: %v", err),
		}
	}

	options = append(options, WithHeader("Content-Type", "application/json"))
	return s.Post(url, bytes.NewReader(jsonData), options...)
}

// Put makes a PUT request using this session
func (s *SessionContext) Put(url string, body io.Reader, options ...RequestOption) (*http.Response, error) {
	return s.Request("PUT", url, body, options...)
}

// Delete makes a DELETE request using this session
func (s *SessionContext) Delete(url string, options ...RequestOption) (*http.Response, error) {
	return s.Request("DELETE", url, nil, options...)
}

// Utility functions for reading responses

// ReadJSON reads and unmarshals JSON response
func ReadJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// ReadString reads response body as string
func ReadString(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ReadBytes reads response body as bytes
func ReadBytes(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Example functions demonstrating usage

func runBasicExamples() {
	fmt.Println("🚀 CycleTLS-Proxy Go Client Examples")
	fmt.Println(strings.Repeat("=", 50))

	client := NewCycleTLSClient("http://localhost:8080")

	// Health check
	fmt.Println("\n📊 Health Check")
	health, err := client.HealthCheck()
	if err != nil {
		fmt.Printf("✗ Health check failed: %v\n", err)
		return
	}

	fmt.Printf("✓ Server status: %v\n", health["status"])
	if proxy, ok := health["proxy"].(map[string]interface{}); ok {
		fmt.Printf("✓ Available profiles: %.0f\n", proxy["profiles_available"])
		fmt.Printf("✓ Active sessions: %.0f\n", proxy["active_sessions"])
	}

	fmt.Println("\n🌐 Basic GET Requests with Different Profiles")

	// Test different browser profiles
	profiles := map[string]string{
		ProfileChrome:    "Chrome Linux",
		ProfileFirefox:   "Firefox Linux",
		ProfileSafariIOS: "Safari iOS",
		ProfileEdge:      "Microsoft Edge",
	}

	for profile, description := range profiles {
		resp, err := client.Get("https://httpbin.org/user-agent", WithProfile(profile))
		if err != nil {
			fmt.Printf("✗ %s: %v\n", description, err)
			continue
		}

		var result map[string]interface{}
		if err := ReadJSON(resp, &result); err != nil {
			fmt.Printf("✗ %s: failed to parse response\n", description)
			continue
		}

		if userAgent, ok := result["user-agent"].(string); ok {
			fmt.Printf("✓ %s: %s\n", description, userAgent)
		}
	}

	fmt.Println("\n📝 HTTP Methods")

	// POST request with JSON
	postData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"active":   true,
		"metadata": map[string]interface{}{
			"source": "go-example",
		},
	}

	resp, err := client.PostJSON("https://httpbin.org/post", postData, WithProfile(ProfileChrome))
	if err != nil {
		fmt.Printf("✗ POST request failed: %v\n", err)
	} else {
		var result map[string]interface{}
		if err := ReadJSON(resp, &result); err == nil {
			if jsonData, ok := result["json"].(map[string]interface{}); ok {
				if username, ok := jsonData["username"].(string); ok {
					fmt.Printf("✓ POST with JSON: %s\n", username)
				}
			}
		}
	}

	// PUT request
	putData := "Updated content"
	resp, err = client.Put("https://httpbin.org/put",
		strings.NewReader(putData),
		WithProfile(ProfileFirefox),
		WithHeader("Content-Type", "text/plain"))
	if err != nil {
		fmt.Printf("✗ PUT request failed: %v\n", err)
	} else {
		fmt.Printf("✓ PUT request: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}

	// DELETE request
	resp, err = client.Delete("https://httpbin.org/delete", WithProfile(ProfileSafari))
	if err != nil {
		fmt.Printf("✗ DELETE request failed: %v\n", err)
	} else {
		fmt.Printf("✓ DELETE request: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}
}

func runSessionExamples() {
	fmt.Println("\n🔄 Session Management")

	client := NewCycleTLSClient("http://localhost:8080")
	session := client.NewSession("demo-session")

	// Set a cookie
	resp, err := session.Get("https://httpbin.org/cookies/set/session_token/abc123")
	if err != nil {
		fmt.Printf("✗ Failed to set cookie: %v\n", err)
		return
	}
	resp.Body.Close()

	// Verify cookie persistence
	resp, err = session.Get("https://httpbin.org/cookies")
	if err != nil {
		fmt.Printf("✗ Failed to get cookies: %v\n", err)
		return
	}

	var result map[string]interface{}
	if err := ReadJSON(resp, &result); err != nil {
		fmt.Printf("✗ Failed to parse cookie response: %v\n", err)
		return
	}

	if cookies, ok := result["cookies"].(map[string]interface{}); ok {
		if token, exists := cookies["session_token"].(string); exists {
			fmt.Printf("✓ Session cookie persisted: %s\n", token)
		} else {
			fmt.Println("✗ Session cookie not found")
		}

		// Add another cookie
		resp, _ = session.Get("https://httpbin.org/cookies/set/user_id/12345")
		if resp != nil {
			resp.Body.Close()
		}

		// Check both cookies
		resp, err = session.Get("https://httpbin.org/cookies")
		if err == nil {
			var result2 map[string]interface{}
			if ReadJSON(resp, &result2) == nil {
				if cookies2, ok := result2["cookies"].(map[string]interface{}); ok {
					cookieNames := make([]string, 0, len(cookies2))
					for name := range cookies2 {
						cookieNames = append(cookieNames, name)
					}
					fmt.Printf("✓ Session has %d cookies: %v\n", len(cookies2), cookieNames)
				}
			}
		}
	}
}

func runAuthenticationExamples() {
	fmt.Println("\n🔐 Authentication Examples")

	client := NewCycleTLSClient("http://localhost:8080")

	// Basic authentication
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	resp, err := client.Get("https://httpbin.org/basic-auth/testuser/testpass",
		WithProfile(ProfileChrome),
		WithHeader("Authorization", "Basic "+credentials))

	if err != nil {
		fmt.Printf("✗ Basic auth failed: %v\n", err)
	} else {
		var result map[string]interface{}
		if err := ReadJSON(resp, &result); err == nil {
			if authenticated, ok := result["authenticated"].(bool); ok {
				fmt.Printf("✓ Basic auth: %v\n", authenticated)
			}
		}
	}

	// Bearer token authentication
	resp, err = client.Get("https://httpbin.org/bearer",
		WithProfile(ProfileFirefox),
		WithHeader("Authorization", "Bearer test-token-12345"))

	if err != nil {
		fmt.Printf("✗ Bearer token failed: %v\n", err)
	} else {
		var result map[string]interface{}
		if err := ReadJSON(resp, &result); err == nil {
			if authenticated, ok := result["authenticated"].(bool); ok {
				fmt.Printf("✓ Bearer token: %v\n", authenticated)
			}
		}
	}
}

func runAdvancedExamples() {
	fmt.Println("\n🚀 Advanced Features")

	client := NewCycleTLSClient("http://localhost:8080")

	// Custom headers
	headers := map[string]string{
		"X-API-Key":        "secret-key-123",
		"X-Client-Version": "1.0.0",
		"Accept":           "application/json",
		"User-Agent":       "This-Should-Be-Overridden/1.0",
	}

	resp, err := client.Get("https://httpbin.org/headers",
		WithProfile(ProfileSafariIOS),
		WithHeaders(headers))

	if err != nil {
		fmt.Printf("✗ Custom headers failed: %v\n", err)
	} else {
		var result map[string]interface{}
		if err := ReadJSON(resp, &result); err == nil {
			if resultHeaders, ok := result["headers"].(map[string]interface{}); ok {
				fmt.Printf("✓ Custom headers forwarded: %d headers\n", len(resultHeaders))

				// Check if User-Agent was correctly overridden by profile
				if userAgent, ok := resultHeaders["User-Agent"].(string); ok && strings.Contains(userAgent, "iPhone") {
					fmt.Println("✓ Profile User-Agent correctly used")
				} else {
					fmt.Println("✗ Custom User-Agent incorrectly used")
				}
			}
		}
	}

	// Timeout configuration
	start := time.Now()
	resp, err = client.Get("https://httpbin.org/delay/1",
		WithProfile(ProfileChrome),
		WithTimeout(5))

	if err != nil {
		fmt.Printf("✗ Timeout test failed: %v\n", err)
	} else {
		duration := time.Since(start)
		fmt.Printf("✓ Timeout test passed: %.2fs\n", duration.Seconds())
		resp.Body.Close()
	}

	// Large JSON payload
	largeData := map[string]interface{}{
		"items": make([]map[string]interface{}, 100),
		"metadata": map[string]interface{}{
			"total":     100,
			"generated": time.Now().Unix(),
			"client":    "go-example",
		},
	}

	// Fill in items
	items := largeData["items"].([]map[string]interface{})
	for i := range items {
		items[i] = map[string]interface{}{
			"id":     i,
			"name":   fmt.Sprintf("Item %d", i),
			"active": true,
		}
	}

	resp, err = client.PostJSON("https://httpbin.org/post", largeData, WithProfile(ProfileEdge))
	if err != nil {
		fmt.Printf("✗ Large JSON payload failed: %v\n", err)
	} else {
		var result map[string]interface{}
		if err := ReadJSON(resp, &result); err == nil {
			if jsonData, ok := result["json"].(map[string]interface{}); ok {
				if items, ok := jsonData["items"].([]interface{}); ok {
					fmt.Printf("✓ Large JSON payload: %d items sent\n", len(items))
				}
			}
		}
	}
}

func runErrorHandlingExamples() {
	fmt.Println("\n⚠️  Error Handling")

	client := NewCycleTLSClient("http://localhost:8080")

	// Invalid profile
	_, err := client.Get("https://httpbin.org/get", WithProfile("invalid-profile"))
	if err != nil {
		var cycleTLSErr *CycleTLSError
		if errors.As(err, &cycleTLSErr) && errors.Is(cycleTLSErr.Type, ErrCycleTLSInvalidProfile) {
			fmt.Println("✓ Invalid profile error handled correctly")
		} else {
			fmt.Printf("✗ Unexpected error for invalid profile: %v\n", err)
		}
	} else {
		fmt.Println("✗ Invalid profile should have failed")
	}

	// Empty URL
	_, err = client.Get("", WithProfile(ProfileChrome))
	if err != nil {
		fmt.Println("✓ Empty URL error handled correctly")
	} else {
		fmt.Println("✗ Empty URL should have failed")
	}

	// Invalid URL
	_, err = client.Get("not-a-url", WithProfile(ProfileChrome))
	if err != nil {
		fmt.Println("✓ Invalid URL error handled correctly")
	} else {
		fmt.Println("✗ Invalid URL should have failed")
	}

	// Timeout error
	_, err = client.Get("https://httpbin.org/delay/5",
		WithProfile(ProfileChrome),
		WithTimeout(1))

	if err != nil {
		var cycleTLSErr *CycleTLSError
		if errors.As(err, &cycleTLSErr) && errors.Is(cycleTLSErr.Type, ErrCycleTLSTimeout) {
			fmt.Println("✓ Timeout error handled correctly")
		} else {
			fmt.Printf("✗ Unexpected timeout error: %v\n", err)
		}
	} else {
		fmt.Println("✗ Timeout should have occurred")
	}
}

func runConcurrentExamples() {
	fmt.Println("\n🔄 Concurrent Requests")

	client := NewCycleTLSClient("http://localhost:8080")

	const numRequests = 10
	var wg sync.WaitGroup
	results := make([]string, numRequests)
	errors := make([]error, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("concurrent-%d", requestID)
			url := fmt.Sprintf("https://httpbin.org/get?request_id=%d", requestID)

			resp, err := client.Get(url,
				WithProfile(ProfileChrome),
				WithSessionID(sessionID))

			if err != nil {
				errors[requestID] = err
				return
			}

			body, err := ReadString(resp)
			if err != nil {
				errors[requestID] = err
				return
			}

			results[requestID] = fmt.Sprintf("Request %d: HTTP %d, %d bytes",
				requestID, resp.StatusCode, len(body))
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	successful := 0
	for i := 0; i < numRequests; i++ {
		if errors[i] == nil {
			successful++
			fmt.Printf("  %s\n", results[i])
		} else {
			fmt.Printf("  Request %d: ERROR - %v\n", i, errors[i])
		}
	}

	fmt.Printf("✓ Completed %d/%d concurrent requests in %.2fs\n",
		successful, numRequests, duration.Seconds())
}

func runRealWorldExamples() {
	fmt.Println("\n🌍 Real-World Examples")

	client := NewCycleTLSClient("http://localhost:8080")

	// Simulate login flow
	fmt.Println("\n🔐 Simulated Login Flow")
	session := client.NewSession("login-demo")

	// Step 1: Get login page (simulate)
	resp, err := session.Get("https://httpbin.org/get", WithProfile(ProfileChrome))
	if err != nil {
		fmt.Printf("✗ Login flow failed at step 1: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("✓ Step 1: Retrieved login page")

	// Step 2: Submit login credentials
	loginData := map[string]interface{}{
		"username":    "demo_user",
		"password":    "secure_password",
		"remember_me": true,
	}

	resp, err = session.PostJSON("https://httpbin.org/post", loginData, WithProfile(ProfileChrome))
	if err != nil {
		fmt.Printf("✗ Login flow failed at step 2: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("✓ Step 2: Submitted login credentials")

	// Step 3: Access protected resource
	resp, err = session.Get("https://httpbin.org/get?protected=true",
		WithProfile(ProfileChrome),
		WithHeader("Authorization", "Bearer fake-jwt-token"))

	if err != nil {
		fmt.Printf("✗ Login flow failed at step 3: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("✓ Step 3: Accessed protected resource")

	// API interaction example
	fmt.Println("\n📡 API Interaction Example")

	// Create resource
	createData := map[string]interface{}{
		"name":        "Test Resource",
		"description": "Created via CycleTLS-Proxy",
		"active":      true,
		"tags":        []string{"test", "example", "go"},
	}

	resp, err = client.PostJSON("https://httpbin.org/post", createData,
		WithProfile(ProfileChrome),
		WithHeader("X-API-Key", "demo-api-key-123"))

	if err != nil {
		fmt.Printf("✗ API create failed: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Println("✓ Created resource via API")
	}

	// Update resource
	updateData := map[string]interface{}{
		"name": "Updated Test Resource",
	}

	resp, err = client.PatchJSON("https://httpbin.org/patch", updateData,
		WithProfile(ProfileChrome),
		WithHeader("X-API-Key", "demo-api-key-123"))

	if err != nil {
		fmt.Printf("✗ API update failed: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Println("✓ Updated resource via API")
	}

	// List resources
	resp, err = client.Get("https://httpbin.org/get?page=1&limit=10",
		WithProfile(ProfileChrome),
		WithHeader("X-API-Key", "demo-api-key-123"))

	if err != nil {
		fmt.Printf("✗ API list failed: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Println("✓ Retrieved resource list via API")
	}
}

func runPerformanceExamples() {
	fmt.Println("\n⚡ Performance Examples")

	client := NewCycleTLSClient("http://localhost:8080")

	// Measure single request latency
	start := time.Now()
	resp, err := client.Get("https://httpbin.org/get", WithProfile(ProfileChrome))
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("✗ Latency test failed: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Printf("✓ Single request latency: %.2fms\n", float64(latency.Nanoseconds())/1e6)
	}

	// Measure throughput with sequential requests
	const requestCount = 20
	session := client.NewSession("throughput-test")

	start = time.Now()
	for i := 0; i < requestCount; i++ {
		url := fmt.Sprintf("https://httpbin.org/get?seq=%d", i)
		resp, err := session.Get(url, WithProfile(ProfileChrome))
		if err != nil {
			fmt.Printf("✗ Throughput test failed at request %d: %v\n", i, err)
			break
		}
		resp.Body.Close()
	}

	duration := time.Since(start)
	throughput := float64(requestCount) / duration.Seconds()
	fmt.Printf("✓ Sequential throughput: %.2f req/s (%d requests in %.2fs)\n",
		throughput, requestCount, duration.Seconds())

	// Test different response sizes
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		start := time.Now()
		url := fmt.Sprintf("https://httpbin.org/drip?duration=0&numbytes=%d", size)
		resp, err := client.Get(url, WithProfile(ProfileChrome))
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("✗ %d bytes test failed: %v\n", size, err)
			continue
		}

		body, err := ReadBytes(resp)
		if err != nil {
			fmt.Printf("✗ %d bytes test failed to read body: %v\n", size, err)
			continue
		}

		fmt.Printf("✓ %d bytes response: %.2fms (actual: %d bytes)\n",
			size, float64(duration.Nanoseconds())/1e6, len(body))
	}
}

func main() {
	fmt.Println("CycleTLS-Proxy Go Client - Comprehensive Examples")
	fmt.Println(strings.Repeat("=", 60))

	// Basic examples
	runBasicExamples()

	// Session management
	runSessionExamples()

	// Authentication
	runAuthenticationExamples()

	// Advanced features
	runAdvancedExamples()

	// Error handling
	runErrorHandlingExamples()

	// Concurrent requests
	runConcurrentExamples()

	// Real-world examples
	runRealWorldExamples()

	// Performance examples
	runPerformanceExamples()

	fmt.Println("\n🎉 All Examples Completed Successfully!")
	fmt.Println("\nTo use this as a library:")
	fmt.Println("```go")
	fmt.Println(`import "path/to/examples"`)
	fmt.Println("")
	fmt.Println(`client := examples.NewCycleTLSClient("http://localhost:8080")`)
	fmt.Println(`resp, err := client.Get("https://api.example.com", examples.WithProfile(examples.ProfileChrome))`)
	fmt.Println("if err != nil {")
	fmt.Println("    log.Fatal(err)")
	fmt.Println("}")
	fmt.Println("defer resp.Body.Close()")
	fmt.Println("```")
}
