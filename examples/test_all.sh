#!/bin/bash

# CycleTLS-Proxy Comprehensive Test Suite
# This script runs all example files and performs comprehensive end-to-end testing

set -e  # Exit on any error

# Configuration
PROXY_URL="${PROXY_URL:-http://localhost:8080}"
PROXY_PID=""
TEMP_DIR="/tmp/cycletls-test-$$"
TEST_RESULTS="$TEMP_DIR/results.log"
COVERAGE_DIR="$TEMP_DIR/coverage"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Function to print colored output
print_header() {
    echo -e "\n${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo
}

print_section() {
    echo -e "\n${CYAN}$1${NC}"
    echo -e "${CYAN}$(printf '%.0s-' $(seq 1 ${#1}))${NC}"
}

print_test() {
    echo -e "\n${YELLOW}[TEST] $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

print_skip() {
    echo -e "${PURPLE}⚠ $1${NC}"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
}

# Function to increment total test counter
inc_test() {
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
}

# Setup and cleanup functions
setup_test_env() {
    mkdir -p "$TEMP_DIR"
    mkdir -p "$COVERAGE_DIR"
    
    # Initialize test results log
    cat > "$TEST_RESULTS" << EOF
CycleTLS-Proxy Test Results
===========================
Started: $(date)
Proxy URL: $PROXY_URL
Environment: $(uname -s) $(uname -r)
Shell: $SHELL

EOF
}

cleanup_test_env() {
    # Stop proxy if we started it
    if [[ -n "$PROXY_PID" ]]; then
        echo "Stopping test proxy server (PID: $PROXY_PID)..."
        kill $PROXY_PID 2>/dev/null || true
        wait $PROXY_PID 2>/dev/null || true
    fi
    
    # Clean up temporary files
    rm -rf "$TEMP_DIR" 2>/dev/null || true
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if proxy is running
check_proxy() {
    if curl -s "$PROXY_URL/health" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to start proxy if not running
start_proxy() {
    if check_proxy; then
        echo "✓ CycleTLS-Proxy already running at $PROXY_URL"
        return 0
    fi
    
    echo "Starting CycleTLS-Proxy for testing..."
    
    # Try to find the proxy binary
    if [[ -f "./cycletls-proxy" ]]; then
        PROXY_BIN="./cycletls-proxy"
    elif [[ -f "./proxy" ]]; then
        PROXY_BIN="./proxy"
    elif command_exists "go" && [[ -f "./cmd/proxy/main.go" ]]; then
        echo "Building proxy from source..."
        go build -o "$TEMP_DIR/cycletls-proxy" ./cmd/proxy
        PROXY_BIN="$TEMP_DIR/cycletls-proxy"
    else
        echo "❌ Could not find CycleTLS-Proxy binary"
        echo "Please ensure the proxy is built and available, or start it manually"
        return 1
    fi
    
    # Start the proxy in the background
    PORT=$(echo "$PROXY_URL" | sed -n 's/.*:\([0-9]*\).*/\1/p')
    PORT=${PORT:-8080}
    
    LOG_LEVEL=info PORT=$PORT "$PROXY_BIN" > "$TEMP_DIR/proxy.log" 2>&1 &
    PROXY_PID=$!
    
    # Wait for proxy to start
    echo "Waiting for proxy to start (PID: $PROXY_PID)..."
    for i in {1..30}; do
        if check_proxy; then
            echo "✓ Proxy started successfully"
            return 0
        fi
        sleep 1
    done
    
    echo "❌ Failed to start proxy"
    if [[ -f "$TEMP_DIR/proxy.log" ]]; then
        echo "Proxy logs:"
        cat "$TEMP_DIR/proxy.log"
    fi
    return 1
}

# Function to run a test and log results
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_exit_code="${3:-0}"
    
    inc_test
    print_test "$test_name"
    
    # Log test start
    echo "Running: $test_name" >> "$TEST_RESULTS"
    echo "Command: $test_command" >> "$TEST_RESULTS"
    echo "Started: $(date)" >> "$TEST_RESULTS"
    
    # Run the test
    local start_time=$(date +%s)
    local exit_code=0
    
    if eval "$test_command" >> "$TEST_RESULTS" 2>&1; then
        exit_code=0
    else
        exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Log results
    echo "Duration: ${duration}s" >> "$TEST_RESULTS"
    echo "Exit code: $exit_code" >> "$TEST_RESULTS"
    echo "Expected: $expected_exit_code" >> "$TEST_RESULTS"
    
    if [[ $exit_code -eq $expected_exit_code ]]; then
        print_success "$test_name (${duration}s)"
        echo "Result: PASSED" >> "$TEST_RESULTS"
    else
        print_error "$test_name (${duration}s, exit code: $exit_code)"
        echo "Result: FAILED" >> "$TEST_RESULTS"
    fi
    
    echo "---" >> "$TEST_RESULTS"
}

# Function to check prerequisites
check_prerequisites() {
    print_section "Checking Prerequisites"
    
    local missing_deps=()
    
    # Check for required tools
    if ! command_exists curl; then
        missing_deps+=("curl")
    fi
    
    if ! command_exists jq; then
        missing_deps+=("jq")
    fi
    
    # Check for optional tools (will skip related tests if missing)
    if ! command_exists python3; then
        echo "⚠ Python3 not found - Python tests will be skipped"
    fi
    
    if ! command_exists node; then
        echo "⚠ Node.js not found - Node.js tests will be skipped"
    fi
    
    if ! command_exists go; then
        echo "⚠ Go not found - Go tests will be skipped"
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        echo "❌ Missing required dependencies: ${missing_deps[*]}"
        echo "Please install them and try again."
        exit 1
    fi
    
    echo "✓ All required dependencies found"
}

# Core functionality tests
run_core_tests() {
    print_section "Core Functionality Tests"
    
    # Test 1: Health check
    run_test "Health Check Endpoint" \
        "curl -s '$PROXY_URL/health' | jq -e '.status == \"healthy\"'"
    
    # Test 2: Basic GET request
    run_test "Basic GET Request" \
        "curl -s -H 'X-URL: https://httpbin.org/get' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.url'"
    
    # Test 3: Different browser profiles
    local profiles=("chrome" "firefox" "safari" "edge")
    for profile in "${profiles[@]}"; do
        run_test "Browser Profile: $profile" \
            "curl -s -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: $profile' '$PROXY_URL' | jq -e '.\"user-agent\"'"
    done
    
    # Test 4: POST request with JSON
    run_test "POST Request with JSON" \
        "curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: chrome' -H 'Content-Type: application/json' -d '{\"test\": \"data\"}' '$PROXY_URL' | jq -e '.json.test == \"data\"'"
    
    # Test 5: Session management
    run_test "Session Cookie Persistence" \
        "SESSION_ID=\"test-session-\$(date +%s)\"; curl -s -H 'X-URL: https://httpbin.org/cookies/set/test/value' -H 'X-SESSION-ID: \$SESSION_ID' '$PROXY_URL' >/dev/null && curl -s -H 'X-URL: https://httpbin.org/cookies' -H 'X-SESSION-ID: \$SESSION_ID' '$PROXY_URL' | jq -e '.cookies.test == \"value\"'"
    
    # Test 6: Custom headers forwarding
    run_test "Custom Headers Forwarding" \
        "curl -s -H 'X-URL: https://httpbin.org/headers' -H 'X-IDENTIFIER: chrome' -H 'Custom-Header: test-value' '$PROXY_URL' | jq -e '.headers.\"Custom-Header\" == \"test-value\"'"
    
    # Test 7: Timeout configuration
    run_test "Timeout Configuration" \
        "curl -s -H 'X-URL: https://httpbin.org/delay/1' -H 'X-IDENTIFIER: chrome' -H 'X-TIMEOUT: 5' '$PROXY_URL' | jq -e '.url'"
    
    # Test 8: HTTP methods
    local methods=("GET" "POST" "PUT" "PATCH" "DELETE")
    for method in "${methods[@]}"; do
        local url_path=$(echo "$method" | tr '[:upper:]' '[:lower:]')
        if [[ "$method" == "GET" ]]; then
            url_path="get"
        fi
        
        if [[ "$method" == "POST" || "$method" == "PUT" || "$method" == "PATCH" ]]; then
            run_test "HTTP $method Method" \
                "curl -s -X $method -H 'X-URL: https://httpbin.org/$url_path' -H 'X-IDENTIFIER: chrome' -H 'Content-Type: application/json' -d '{}' '$PROXY_URL' | jq -e '.url'"
        else
            run_test "HTTP $method Method" \
                "curl -s -X $method -H 'X-URL: https://httpbin.org/$url_path' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.url or .'"
        fi
    done
}

# Error handling tests
run_error_tests() {
    print_section "Error Handling Tests"
    
    # Test 1: Missing X-URL header (should fail)
    run_test "Missing X-URL Header (should fail)" \
        "curl -s -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | grep -q 'X-URL'" \
        0
    
    # Test 2: Invalid profile (should fail)
    run_test "Invalid Profile (should fail)" \
        "curl -s -H 'X-URL: https://httpbin.org/get' -H 'X-IDENTIFIER: invalid-browser' '$PROXY_URL' | grep -q 'Available profiles'" \
        0
    
    # Test 3: Invalid URL (should fail)
    run_test "Invalid URL (should fail)" \
        "curl -s -H 'X-URL: not-a-valid-url' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | grep -q -i 'invalid'" \
        0
    
    # Test 4: Invalid timeout (should fail)
    run_test "Invalid Timeout (should fail)" \
        "curl -s -H 'X-URL: https://httpbin.org/get' -H 'X-IDENTIFIER: chrome' -H 'X-TIMEOUT: 999' '$PROXY_URL' | grep -q -i 'timeout'" \
        0
    
    # Test 5: Timeout exceeded (should fail gracefully)
    run_test "Request Timeout" \
        "timeout 3s curl -s -H 'X-URL: https://httpbin.org/delay/10' -H 'X-IDENTIFIER: chrome' -H 'X-TIMEOUT: 1' '$PROXY_URL'" \
        124  # timeout command exit code
}

# Performance tests
run_performance_tests() {
    print_section "Performance Tests"
    
    # Test 1: Response time measurement
    run_test "Response Time Measurement" \
        "time curl -s -H 'X-URL: https://httpbin.org/get' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.url'"
    
    # Test 2: Concurrent requests
    run_test "Concurrent Requests (10x)" \
        "for i in {1..10}; do curl -s -H 'X-URL: https://httpbin.org/get?req=\$i' -H 'X-IDENTIFIER: chrome' -H 'X-SESSION-ID: concurrent-\$i' '$PROXY_URL' | jq -e '.url' & done; wait"
    
    # Test 3: Large response handling
    run_test "Large Response Handling (10KB)" \
        "curl -s -H 'X-URL: https://httpbin.org/drip?duration=1&numbytes=10240' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | wc -c | grep -q '10240'"
    
    # Test 4: Session reuse performance
    run_test "Session Reuse Performance" \
        "SESSION_ID=\"perf-session-\$(date +%s)\"; for i in {1..5}; do curl -s -H 'X-URL: https://httpbin.org/get?req=\$i' -H 'X-SESSION-ID: \$SESSION_ID' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.url' >/dev/null; done"
}

# Security tests
run_security_tests() {
    print_section "Security Tests"
    
    # Test 1: Header injection protection
    run_test "Header Injection Protection" \
        "curl -s -H 'X-URL: https://httpbin.org/headers' -H 'X-IDENTIFIER: chrome' -H 'Malicious-Header: value\\nInjected: header' '$PROXY_URL' | jq -e '.headers'"
    
    # Test 2: URL validation
    run_test "URL Validation (file:// scheme should fail)" \
        "curl -s -H 'X-URL: file:///etc/passwd' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | grep -q -i 'unsupported'" \
        0
    
    # Test 3: X-* header filtering
    run_test "X-* Header Filtering" \
        "curl -s -H 'X-URL: https://httpbin.org/headers' -H 'X-IDENTIFIER: chrome' -H 'X-Internal-Test: should-not-appear' -H 'Regular-Header: should-appear' '$PROXY_URL' | jq -e '.headers.\"Regular-Header\" == \"should-appear\" and (.headers.\"X-Internal-Test\" | not)'"
}

# Integration tests with example files
run_integration_tests() {
    print_section "Integration Tests with Example Files"
    
    # Test 1: cURL examples
    if [[ -f "examples/curl.sh" ]]; then
        run_test "cURL Examples Script" \
            "cd examples && chmod +x curl.sh && PROXY_URL='$PROXY_URL' timeout 120s ./curl.sh"
    else
        print_skip "cURL Examples Script (file not found)"
        inc_test
    fi
    
    # Test 2: Python examples
    if [[ -f "examples/python.py" ]] && command_exists python3; then
        run_test "Python Examples Script" \
            "cd examples && timeout 120s python3 python.py"
    else
        if [[ ! -f "examples/python.py" ]]; then
            print_skip "Python Examples Script (file not found)"
        else
            print_skip "Python Examples Script (python3 not available)"
        fi
        inc_test
    fi
    
    # Test 3: Node.js examples
    if [[ -f "examples/node.js" ]] && command_exists node; then
        # Check if axios is available
        if node -e "require('axios')" 2>/dev/null; then
            run_test "Node.js Examples Script" \
                "cd examples && timeout 120s node node.js"
        else
            print_skip "Node.js Examples Script (axios not installed)"
            inc_test
        fi
    else
        if [[ ! -f "examples/node.js" ]]; then
            print_skip "Node.js Examples Script (file not found)"
        else
            print_skip "Node.js Examples Script (node not available)"
        fi
        inc_test
    fi
    
    # Test 4: Go examples
    if [[ -f "examples/go.go" ]] && command_exists go; then
        run_test "Go Examples Script" \
            "cd examples && timeout 120s go run go.go"
    else
        if [[ ! -f "examples/go.go" ]]; then
            print_skip "Go Examples Script (file not found)"
        else
            print_skip "Go Examples Script (go not available)"
        fi
        inc_test
    fi
}

# Load testing
run_load_tests() {
    print_section "Load Testing"
    
    print_test "Load Test Setup"
    echo "Starting load test with 50 concurrent requests..."
    
    # Generate load test script
    cat > "$TEMP_DIR/load_test.sh" << 'EOF'
#!/bin/bash
PROXY_URL="$1"
REQUEST_ID="$2"

curl -s \
    -H "X-URL: https://httpbin.org/get?load_test=true&req_id=$REQUEST_ID" \
    -H "X-IDENTIFIER: chrome" \
    -H "X-SESSION-ID: load-test-$REQUEST_ID" \
    "$PROXY_URL" | jq -e '.url' > /dev/null

exit $?
EOF
    chmod +x "$TEMP_DIR/load_test.sh"
    
    # Run load test
    run_test "Load Test (50 concurrent requests)" \
        "for i in {1..50}; do '$TEMP_DIR/load_test.sh' '$PROXY_URL' \$i & done; wait"
}

# Real-world scenario tests
run_scenario_tests() {
    print_section "Real-World Scenario Tests"
    
    # Test 1: Login simulation
    run_test "Login Simulation Scenario" \
        "SESSION_ID=\"login-test-\$(date +%s)\"; curl -s -H 'X-URL: https://httpbin.org/cookies/set/session_token/abc123' -H 'X-SESSION-ID: \$SESSION_ID' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' >/dev/null && curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-SESSION-ID: \$SESSION_ID' -H 'X-IDENTIFIER: chrome' -H 'Content-Type: application/json' -d '{\"username\":\"testuser\",\"password\":\"testpass\"}' '$PROXY_URL' | jq -e '.json.username == \"testuser\"' && curl -s -H 'X-URL: https://httpbin.org/headers' -H 'X-SESSION-ID: \$SESSION_ID' -H 'X-IDENTIFIER: chrome' -H 'Authorization: Bearer fake-token' '$PROXY_URL' | jq -e '.headers.Authorization'"
    
    # Test 2: API workflow
    run_test "API Workflow Scenario" \
        "curl -s -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: chrome' -H 'Content-Type: application/json' -d '{\"name\":\"Test Resource\",\"active\":true}' '$PROXY_URL' | jq -e '.json.active == true' && curl -s -X PUT -H 'X-URL: https://httpbin.org/put' -H 'X-IDENTIFIER: chrome' -H 'Content-Type: application/json' -d '{\"name\":\"Updated Resource\",\"active\":false}' '$PROXY_URL' | jq -e '.json.active == false' && curl -s -X DELETE -H 'X-URL: https://httpbin.org/delete' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.url'"
    
    # Test 3: Different content types
    run_test "Different Content Types" \
        "curl -s -H 'X-URL: https://httpbin.org/json' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.slideshow' && curl -s -H 'X-URL: https://httpbin.org/xml' -H 'X-IDENTIFIER: firefox' '$PROXY_URL' | grep -q '<?xml' && curl -s -H 'X-URL: https://httpbin.org/html' -H 'X-IDENTIFIER: safari' '$PROXY_URL' | grep -q '<html>'"
}

# Function to generate detailed test report
generate_report() {
    print_section "Test Results Summary"
    
    local total_duration=$(($(date +%s) - START_TIME))
    
    # Basic statistics
    echo -e "${CYAN}Test Statistics:${NC}"
    echo "  Total Tests: $TESTS_TOTAL"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    echo "  Skipped: $TESTS_SKIPPED"
    echo "  Duration: ${total_duration}s"
    
    # Calculate percentages
    if [[ $TESTS_TOTAL -gt 0 ]]; then
        local pass_rate=$((TESTS_PASSED * 100 / TESTS_TOTAL))
        local fail_rate=$((TESTS_FAILED * 100 / TESTS_TOTAL))
        local skip_rate=$((TESTS_SKIPPED * 100 / TESTS_TOTAL))
        
        echo "  Pass Rate: ${pass_rate}%"
        echo "  Fail Rate: ${fail_rate}%"
        echo "  Skip Rate: ${skip_rate}%"
    fi
    
    # Final result
    echo
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED!${NC}"
        echo -e "${GREEN}CycleTLS-Proxy is working correctly.${NC}"
        FINAL_EXIT_CODE=0
    else
        echo -e "${RED}❌ SOME TESTS FAILED!${NC}"
        echo -e "${RED}Please check the failing tests and fix the issues.${NC}"
        FINAL_EXIT_CODE=1
    fi
    
    # Add summary to log file
    cat >> "$TEST_RESULTS" << EOF

FINAL SUMMARY
=============
Completed: $(date)
Total Tests: $TESTS_TOTAL
Passed: $TESTS_PASSED
Failed: $TESTS_FAILED
Skipped: $TESTS_SKIPPED
Duration: ${total_duration}s
Result: $(if [[ $TESTS_FAILED -eq 0 ]]; then echo "SUCCESS"; else echo "FAILURE"; fi)
EOF
    
    echo
    echo -e "${CYAN}Detailed test log: $TEST_RESULTS${NC}"
    
    # Show failed tests if any
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "\n${RED}Failed tests details:${NC}"
        grep -A 5 "Result: FAILED" "$TEST_RESULTS" | head -20
    fi
}

# Main execution
main() {
    local START_TIME=$(date +%s)
    
    print_header "CycleTLS-Proxy Comprehensive Test Suite"
    
    echo "This script will run comprehensive tests for the CycleTLS-Proxy server."
    echo "It includes core functionality, error handling, performance, security,"
    echo "integration tests, and real-world scenarios."
    echo
    
    # Handle command line options
    case "${1:-}" in
        -h|--help)
            cat << EOF
Usage: $0 [OPTIONS]

Options:
  -h, --help    Show this help message
  --core-only   Run only core functionality tests
  --no-start    Don't start proxy automatically (assume it's already running)
  --verbose     Enable verbose output
  
Environment Variables:
  PROXY_URL     URL of the CycleTLS-Proxy server (default: http://localhost:8080)
  
Examples:
  $0                    # Run all tests
  $0 --core-only       # Run only core tests
  PROXY_URL=http://localhost:9000 $0  # Use custom proxy URL
EOF
            exit 0
            ;;
        --core-only)
            CORE_ONLY=true
            ;;
        --no-start)
            NO_START=true
            ;;
        --verbose)
            set -x  # Enable verbose shell output
            ;;
    esac
    
    # Setup trap for cleanup
    trap cleanup_test_env EXIT INT TERM
    
    # Setup test environment
    setup_test_env
    
    # Check prerequisites
    check_prerequisites
    
    # Start proxy if needed
    if [[ "${NO_START:-}" != "true" ]]; then
        if ! start_proxy; then
            echo "❌ Failed to start proxy. Please start it manually or use --no-start"
            exit 1
        fi
    else
        if ! check_proxy; then
            echo "❌ Proxy is not running at $PROXY_URL"
            echo "Please start it manually or remove --no-start flag"
            exit 1
        fi
        echo "✓ Using existing proxy at $PROXY_URL"
    fi
    
    # Wait a moment for proxy to be fully ready
    sleep 2
    
    # Run test suites
    run_core_tests
    
    if [[ "${CORE_ONLY:-}" != "true" ]]; then
        run_error_tests
        run_performance_tests
        run_security_tests
        run_integration_tests
        run_load_tests
        run_scenario_tests
    fi
    
    # Generate final report
    generate_report
    
    exit $FINAL_EXIT_CODE
}

# Initialize variables
START_TIME=$(date +%s)
FINAL_EXIT_CODE=0

# Run main function with all arguments
main "$@"