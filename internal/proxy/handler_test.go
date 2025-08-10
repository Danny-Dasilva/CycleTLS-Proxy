package proxy

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Danny-Dasilva/CycleTLS-Proxy/internal/fingerprints"
	"github.com/charmbracelet/log"
	"github.com/valyala/fasthttp"
)

func TestNewHandler(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.logger == nil {
		t.Error("Handler logger is nil")
	}

	if handler.clients == nil {
		t.Error("Handler clients map is nil")
	}

	if handler.profiles == nil {
		t.Error("Handler profiles map is nil")
	}

	if handler.defaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", handler.defaultTimeout)
	}
}

func TestExtractHeaders(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	tests := []struct {
		name        string
		headers     map[string]string
		expected    *RequestHeaders
		expectError bool
	}{
		{
			name: "valid headers with all fields",
			headers: map[string]string{
				"X-URL":        "https://example.com",
				"X-IDENTIFIER": "firefox",
				"X-SESSION-ID": "test-session",
				"X-PROXY":      "http://proxy:8080",
				"X-TIMEOUT":    "60",
			},
			expected: &RequestHeaders{
				TargetURL:  "https://example.com",
				Identifier: "firefox",
				SessionID:  "test-session",
				Proxy:      "http://proxy:8080",
				Timeout:    60 * time.Second,
			},
			expectError: false,
		},
		{
			name: "minimal valid headers",
			headers: map[string]string{
				"X-URL": "https://example.com",
			},
			expected: &RequestHeaders{
				TargetURL:  "https://example.com",
				Identifier: "chrome", // default
				SessionID:  "",
				Proxy:      "",
				Timeout:    30 * time.Second, // default
			},
			expectError: false,
		},
		{
			name:        "missing X-URL",
			headers:     map[string]string{},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid timeout",
			headers: map[string]string{
				"X-URL":     "https://example.com",
				"X-TIMEOUT": "invalid",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "timeout too high",
			headers: map[string]string{
				"X-URL":     "https://example.com",
				"X-TIMEOUT": "500",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "timeout too low",
			headers: map[string]string{
				"X-URL":     "https://example.com",
				"X-TIMEOUT": "0",
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock fasthttp context
			ctx := &fasthttp.RequestCtx{}

			// Set headers
			for key, value := range tt.headers {
				ctx.Request.Header.Set(key, value)
			}

			// Extract headers
			result, err := handler.extractHeaders(ctx)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// If expecting error, we're done
			if tt.expectError {
				return
			}

			// Compare results
			if result.TargetURL != tt.expected.TargetURL {
				t.Errorf("TargetURL: expected %s, got %s", tt.expected.TargetURL, result.TargetURL)
			}
			if result.Identifier != tt.expected.Identifier {
				t.Errorf("Identifier: expected %s, got %s", tt.expected.Identifier, result.Identifier)
			}
			if result.SessionID != tt.expected.SessionID {
				t.Errorf("SessionID: expected %s, got %s", tt.expected.SessionID, result.SessionID)
			}
			if result.Proxy != tt.expected.Proxy {
				t.Errorf("Proxy: expected %s, got %s", tt.expected.Proxy, result.Proxy)
			}
			if result.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout: expected %v, got %v", tt.expected.Timeout, result.Timeout)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid https URL",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:        "valid http URL",
			url:         "http://example.com",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			url:         "https://example.com/api/v1/test",
			expectError: false,
		},
		{
			name:        "valid URL with query params",
			url:         "https://example.com?param=value",
			expectError: false,
		},
		{
			name:        "invalid scheme ftp",
			url:         "ftp://example.com",
			expectError: true,
		},
		{
			name:        "invalid scheme file",
			url:         "file:///etc/passwd",
			expectError: true,
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "URL without host",
			url:         "https://",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateURL(tt.url)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for URL: %s", tt.url)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for URL %s: %v", tt.url, err)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	// Test getting a valid profile
	profile, err := handler.getProfile("chrome")
	if err != nil {
		t.Errorf("Expected to find chrome profile, got error: %v", err)
	}
	if profile.JA3 == "" && profile.JA4 == "" {
		t.Error("Profile should have either JA3 or JA4 fingerprint")
	}

	// Test getting an invalid profile
	_, err = handler.getProfile("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent profile")
	}
}

func TestGetAvailableProfiles(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	profiles := handler.GetAvailableProfiles()
	if len(profiles) == 0 {
		t.Error("Expected at least one available profile")
	}

	// Check that default profiles are included
	defaultProfiles := fingerprints.GetDefaultProfiles()
	if len(profiles) != len(defaultProfiles) {
		t.Errorf("Expected %d profiles, got %d", len(defaultProfiles), len(profiles))
	}
}

func TestSetDefaultTimeout(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	newTimeout := 45 * time.Second
	handler.SetDefaultTimeout(newTimeout)

	if handler.defaultTimeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, handler.defaultTimeout)
	}
}

func TestSendError(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	ctx := &fasthttp.RequestCtx{}
	handler.sendError(ctx, fasthttp.StatusBadRequest, "Test error message")

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}

	body := string(ctx.Response.Body())
	if body != "Test error message" {
		t.Errorf("Expected body 'Test error message', got '%s'", body)
	}
}

func TestHandleHealthCheck(t *testing.T) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/health")

	handler.HandleRequest(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusOK, ctx.Response.StatusCode())
	}

	contentType := string(ctx.Response.Header.Peek("Content-Type"))
	if contentType != "application/json" {
		t.Errorf("Expected content-type 'application/json', got '%s'", contentType)
	}

	body := string(ctx.Response.Body())
	if body == "" {
		t.Error("Expected non-empty response body")
	}

	// Basic JSON validation
	if !strings.Contains(body, `"status": "healthy"`) {
		t.Errorf("Expected health response to contain status healthy, got: %s", body)
	}
}

// Benchmark tests for performance analysis
func BenchmarkExtractHeaders(b *testing.B) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("X-URL", "https://example.com")
	ctx.Request.Header.Set("X-IDENTIFIER", "chrome")
	ctx.Request.Header.Set("X-SESSION-ID", "test-session")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.extractHeaders(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateURL(b *testing.B) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)
	url := "https://example.com/api/v1/test?param=value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := handler.validateURL(url)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetProfile(b *testing.B) {
	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.getProfile("chrome")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test helper functions
func createTestContext(method, url string, headers map[string]string, body string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	
	// Set method
	ctx.Request.Header.SetMethod(method)
	
	// Set headers
	for key, value := range headers {
		ctx.Request.Header.Set(key, value)
	}
	
	// Set body
	ctx.Request.SetBodyString(body)
	
	return ctx
}

// Example of integration-style test (commented out since it requires network access)
/*
func TestHandleRequest_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := log.New(io.Discard)
	handler := NewHandler(logger)

	ctx := createTestContext("GET", "", map[string]string{
		"X-URL":        "https://httpbin.org/get",
		"X-IDENTIFIER": "chrome",
	}, "")

	// This would require actual network access
	handler.HandleRequest(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
	}
}
*/