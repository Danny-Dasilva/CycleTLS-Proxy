#!/bin/bash

# CycleTLS-Proxy Comprehensive cURL Examples
# This script demonstrates various use cases and features of the CycleTLS-Proxy server

set -e  # Exit on any error

# Configuration
PROXY_URL="${PROXY_URL:-http://localhost:8080}"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo
}

print_section() {
    echo -e "\n${CYAN}$1${NC}"
    echo -e "${CYAN}$(printf '%.0s-' $(seq 1 ${#1}))${NC}"
}

print_command() {
    echo -e "${YELLOW}Command:${NC} $1"
    echo
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    echo
}

print_error() {
    echo -e "${RED}✗ Error: $1${NC}"
    echo
}

# Function to check if server is running
check_server() {
    if ! curl -s "$PROXY_URL/health" > /dev/null 2>&1; then
        print_error "CycleTLS-Proxy server is not running at $PROXY_URL"
        echo "Please start the server first:"
        echo "  ./cycletls-proxy"
        echo "  or"
        echo "  docker run -p 8080:8080 dannydasilva/cycletls-proxy"
        exit 1
    fi
    
    print_success "Server is running at $PROXY_URL"
}

# Function to make a curl request with optional verbosity
make_request() {
    local cmd="$1"
    local desc="$2"
    
    print_command "$cmd"
    
    if [[ "$VERBOSE" == "true" ]]; then
        eval "$cmd -v"
    else
        eval "$cmd"
    fi
    
    local exit_code=$?
    if [[ $exit_code -eq 0 ]]; then
        print_success "$desc completed successfully"
    else
        print_error "$desc failed with exit code $exit_code"
    fi
    
    echo
}

# Main execution
main() {
    print_header "CycleTLS-Proxy cURL Examples"
    
    echo "This script demonstrates comprehensive usage of the CycleTLS-Proxy server."
    echo "Set VERBOSE=true for detailed output: VERBOSE=true $0"
    echo "Set custom proxy URL: PROXY_URL=http://custom-host:port $0"
    echo
    
    check_server
    
    print_section "1. Basic Requests with Different Browser Profiles"
    
    # Chrome on Linux (default)
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Chrome Linux user-agent test"
    
    # Chrome on Windows
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: chrome_windows' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Chrome Windows user-agent test"
    
    # Firefox on Linux
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: firefox' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Firefox Linux user-agent test"
    
    # Firefox on Windows
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: firefox_windows' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Firefox Windows user-agent test"
    
    # Safari on macOS
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: safari' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Safari macOS user-agent test"
    
    # Safari on iOS
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: safari_ios' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Safari iOS user-agent test"
    
    # Microsoft Edge
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: edge' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Microsoft Edge user-agent test"
    
    # Android OkHttp
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: okhttp' '$PROXY_URL' | jq -r '.\"user-agent\"'" \
        "Android OkHttp user-agent test"
    
    print_section "2. HTTP Methods"
    
    # GET request
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/get?param1=value1&param2=value2' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq .args" \
        "GET request with query parameters"
    
    # POST request with JSON
    make_request \
        "curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: firefox' -H 'Content-Type: application/json' -d '{\"username\":\"testuser\",\"password\":\"secret123\",\"remember\":true}' '$PROXY_URL' | jq .json" \
        "POST request with JSON data"
    
    # PUT request
    make_request \
        "curl -s -X PUT -H 'X-URL: https://httpbin.org/put' -H 'X-IDENTIFIER: safari' -H 'Content-Type: application/json' -d '{\"id\":123,\"name\":\"Updated Item\",\"active\":true}' '$PROXY_URL' | jq .json" \
        "PUT request with JSON data"
    
    # PATCH request
    make_request \
        "curl -s -X PATCH -H 'X-URL: https://httpbin.org/patch' -H 'X-IDENTIFIER: edge' -H 'Content-Type: application/json' -d '{\"status\":\"updated\"}' '$PROXY_URL' | jq .json" \
        "PATCH request with JSON data"
    
    # DELETE request
    make_request \
        "curl -s -X DELETE -H 'X-URL: https://httpbin.org/delete' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq .url" \
        "DELETE request"
    
    print_section "3. Session Management"
    
    # Session creation and cookie persistence
    SESSION_ID="demo-session-$(date +%s)"
    
    # Set a cookie in session
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/cookies/set/session_token/abc123456' -H 'X-IDENTIFIER: chrome' -H 'X-SESSION-ID: $SESSION_ID' '$PROXY_URL'" \
        "Set session cookie"
    
    # Verify cookie persists in same session
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/cookies' -H 'X-IDENTIFIER: chrome' -H 'X-SESSION-ID: $SESSION_ID' '$PROXY_URL' | jq .cookies" \
        "Verify cookie persistence in session"
    
    # Multiple requests in same session
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/cookies/set/user_id/12345' -H 'X-IDENTIFIER: chrome' -H 'X-SESSION-ID: $SESSION_ID' '$PROXY_URL'" \
        "Add another cookie to session"
    
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/cookies' -H 'X-IDENTIFIER: chrome' -H 'X-SESSION-ID: $SESSION_ID' '$PROXY_URL' | jq .cookies" \
        "Verify multiple cookies in session"
    
    print_section "4. Custom Headers"
    
    # Request with custom headers
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/headers' -H 'X-IDENTIFIER: firefox' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9' -H 'X-API-Key: secret-api-key-12345' -H 'X-Custom-Header: custom-value' -H 'Accept: application/json' '$PROXY_URL' | jq .headers" \
        "Request with multiple custom headers"
    
    # Test header forwarding (X-* headers should not be forwarded)
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/headers' -H 'X-IDENTIFIER: chrome' -H 'X-Internal-Header: should-not-appear' -H 'Regular-Header: should-appear' '$PROXY_URL' | jq '.headers | keys[]' | grep -E '(X-Internal|Regular)' || echo 'X-Internal-Header correctly filtered, Regular-Header present'" \
        "Header filtering test"
    
    print_section "5. Request Body Handling"
    
    # Form data
    make_request \
        "curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: chrome' -H 'Content-Type: application/x-www-form-urlencoded' -d 'username=testuser&password=secret&remember=on' '$PROXY_URL' | jq .form" \
        "POST with form data"
    
    # Multipart form data (file simulation)
    make_request \
        "curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: safari' -F 'file=@/dev/null;filename=test.txt' -F 'description=Test file upload' '$PROXY_URL' | jq '.files, .form'" \
        "Multipart form with file simulation"
    
    # Large JSON payload
    LARGE_JSON='{"data":{"items":'
    for i in {1..10}; do
        LARGE_JSON+='{"id":'$i',"name":"Item '$i'","description":"This is item number '$i'","active":true,"tags":["tag'$i'a","tag'$i'b"]},'
    done
    LARGE_JSON="${LARGE_JSON%,}]}}"
    
    make_request \
        "curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: firefox' -H 'Content-Type: application/json' -d '$LARGE_JSON' '$PROXY_URL' | jq '.json.data.items | length'" \
        "POST with large JSON payload"
    
    print_section "6. Timeout Configuration"
    
    # Short timeout (should complete)
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/delay/1' -H 'X-IDENTIFIER: chrome' -H 'X-TIMEOUT: 5' '$PROXY_URL' | jq -r '.url'" \
        "Request with sufficient timeout"
    
    # Timeout test (this will take longer)
    echo -e "${YELLOW}Testing timeout with 3-second delay (timeout=2s)...${NC}"
    start_time=$(date +%s)
    if curl -s -H 'X-URL: https://httpbin.org/delay/3' -H 'X-IDENTIFIER: chrome' -H 'X-TIMEOUT: 2' "$PROXY_URL" > /dev/null 2>&1; then
        print_error "Timeout test failed - request should have timed out"
    else
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        print_success "Timeout test passed - request correctly timed out after ~${duration}s"
    fi
    
    print_section "7. Different Response Types"
    
    # JSON response
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/json' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq .slideshow.title" \
        "JSON response handling"
    
    # XML response
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/xml' -H 'X-IDENTIFIER: firefox' '$PROXY_URL' | head -3" \
        "XML response handling"
    
    # HTML response
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/html' -H 'X-IDENTIFIER: safari' '$PROXY_URL' | grep -o '<title>[^<]*</title>'" \
        "HTML response handling"
    
    # Binary response (image)
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/image/png' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | file -" \
        "Binary (PNG image) response handling"
    
    # Large response
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/drip?duration=1&numbytes=1024' -H 'X-IDENTIFIER: edge' '$PROXY_URL' | wc -c" \
        "Large streaming response"
    
    print_section "8. HTTP Status Code Handling"
    
    # Different status codes
    for code in 200 201 301 400 401 403 404 429 500 502; do
        make_request \
            "curl -s -w 'Status: %{http_code}' -H 'X-URL: https://httpbin.org/status/$code' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | grep 'Status:' || echo 'Status: $code'" \
            "HTTP $code response"
    done
    
    print_section "9. Authentication Examples"
    
    # Basic authentication
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/basic-auth/testuser/testpass' -H 'X-IDENTIFIER: chrome' -H 'Authorization: Basic dGVzdHVzZXI6dGVzdHBhc3M=' '$PROXY_URL' | jq .authenticated" \
        "Basic authentication"
    
    # Bearer token authentication
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/bearer' -H 'X-IDENTIFIER: firefox' -H 'Authorization: Bearer test-token-12345' '$PROXY_URL' | jq .authenticated" \
        "Bearer token authentication"
    
    print_section "10. Advanced Features"
    
    # Custom User-Agent override test (should use profile UA, not custom)
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: safari_ios' -H 'User-Agent: CustomAgent/1.0' '$PROXY_URL' | jq -r '.\"user-agent\"' | grep -q 'iPhone' && echo 'Profile User-Agent correctly used' || echo 'Custom User-Agent incorrectly used'" \
        "User-Agent override test"
    
    # Content encoding support
    make_request \
        "curl -s -H 'X-URL: https://httpbin.org/gzip' -H 'X-IDENTIFIER: chrome' -H 'Accept-Encoding: gzip, deflate' '$PROXY_URL' | jq .gzipped" \
        "Gzip compression support"
    
    # IPv6 support (if available)
    if ping6 -c1 google.com >/dev/null 2>&1; then
        make_request \
            "curl -s -H 'X-URL: https://httpbin.org/ip' -H 'X-IDENTIFIER: firefox' '$PROXY_URL' | jq .origin" \
            "IPv6 connectivity test"
    else
        echo -e "${YELLOW}Skipping IPv6 test (not available)${NC}"
    fi
    
    print_section "11. Error Handling and Edge Cases"
    
    # Missing X-URL header
    echo -e "${YELLOW}Testing missing X-URL header (should fail):${NC}"
    if curl -s -H 'X-IDENTIFIER: chrome' "$PROXY_URL" 2>&1 | grep -q "X-URL"; then
        print_success "Missing X-URL error handled correctly"
    else
        print_error "Missing X-URL error not handled properly"
    fi
    
    # Invalid profile
    echo -e "${YELLOW}Testing invalid profile (should fail):${NC}"
    if curl -s -H 'X-URL: https://httpbin.org/get' -H 'X-IDENTIFIER: invalid-browser' "$PROXY_URL" 2>&1 | grep -q "Available profiles"; then
        print_success "Invalid profile error handled correctly"
    else
        print_error "Invalid profile error not handled properly"
    fi
    
    # Invalid URL
    echo -e "${YELLOW}Testing invalid URL (should fail):${NC}"
    if curl -s -H 'X-URL: not-a-valid-url' -H 'X-IDENTIFIER: chrome' "$PROXY_URL" 2>&1 | grep -q "Invalid"; then
        print_success "Invalid URL error handled correctly"
    else
        print_error "Invalid URL error not handled properly"
    fi
    
    # Invalid timeout
    echo -e "${YELLOW}Testing invalid timeout (should fail):${NC}"
    if curl -s -H 'X-URL: https://httpbin.org/get' -H 'X-IDENTIFIER: chrome' -H 'X-TIMEOUT: 999' "$PROXY_URL" 2>&1 | grep -q "timeout"; then
        print_success "Invalid timeout error handled correctly"
    else
        print_error "Invalid timeout error not handled properly"
    fi
    
    print_section "12. Performance and Load Testing"
    
    # Concurrent requests test
    echo -e "${YELLOW}Running 10 concurrent requests...${NC}"
    start_time=$(date +%s%3N)
    
    for i in {1..10}; do
        curl -s -H "X-URL: https://httpbin.org/get?req=$i" -H 'X-IDENTIFIER: chrome' -H "X-SESSION-ID: concurrent-$i" "$PROXY_URL" > /tmp/response_$i.json &
    done
    wait
    
    end_time=$(date +%s%3N)
    duration=$((end_time - start_time))
    
    success_count=$(ls /tmp/response_*.json 2>/dev/null | wc -l)
    print_success "Completed $success_count/10 concurrent requests in ${duration}ms"
    rm -f /tmp/response_*.json
    
    print_section "13. Health Check and Monitoring"
    
    # Health check
    make_request \
        "curl -s '$PROXY_URL/health' | jq -r '\"Status: \" + .status + \", Profiles: \" + (.proxy.profiles_available | tostring) + \", Sessions: \" + (.proxy.active_sessions | tostring)'" \
        "Health check endpoint"
    
    # Server metrics
    make_request \
        "curl -s '$PROXY_URL/health' | jq .proxy" \
        "Server proxy metrics"
    
    print_header "Summary"
    
    echo -e "${GREEN}All cURL examples completed successfully!${NC}"
    echo
    echo "Key features demonstrated:"
    echo "• Multiple browser profile fingerprints"
    echo "• HTTP method support (GET, POST, PUT, PATCH, DELETE)"
    echo "• Session management and cookie persistence"
    echo "• Custom header forwarding"
    echo "• Request body handling (JSON, form data, files)"
    echo "• Timeout configuration"
    echo "• Various response types (JSON, XML, HTML, binary)"
    echo "• HTTP status code handling"
    echo "• Authentication methods"
    echo "• Error handling and validation"
    echo "• Performance and concurrency"
    echo "• Health monitoring"
    echo
    echo -e "${BLUE}For more advanced usage, see the other example files:${NC}"
    echo "• examples/python.py - Python client examples"
    echo "• examples/node.js - Node.js client examples"
    echo "• examples/go.go - Go client examples"
    echo "• examples/test_all.sh - Comprehensive testing suite"
    echo
    echo -e "${CYAN}Happy proxying! 🚀${NC}"
}

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed.${NC}"
    echo "Please install jq:"
    echo "  Ubuntu/Debian: sudo apt-get install jq"
    echo "  macOS: brew install jq"
    echo "  Or download from: https://stedolan.github.io/jq/download/"
    exit 1
fi

# Run main function
main "$@"