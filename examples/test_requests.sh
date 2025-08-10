#!/bin/bash

# Test script to demonstrate CycleTLS-Proxy functionality
# Run this after starting the proxy server with: go run ./cmd/proxy/main.go

PROXY_URL="http://localhost:8080"

echo "🧪 Testing CycleTLS-Proxy Server"
echo "================================="
echo

echo "1. Testing GET request with Chrome profile..."
curl -s -H "X-URL: https://httpbin.org/get" \
     -H "X-IDENTIFIER: chrome" \
     -H "X-SESSION-ID: test-session-1" \
     "$PROXY_URL" | jq .
echo

echo "2. Testing GET request with Firefox profile..."
curl -s -H "X-URL: https://httpbin.org/get" \
     -H "X-IDENTIFIER: firefox" \
     -H "X-SESSION-ID: test-session-2" \
     "$PROXY_URL" | jq .
echo

echo "3. Testing GET request with custom timeout..."
curl -s -H "X-URL: https://httpbin.org/delay/2" \
     -H "X-IDENTIFIER: chrome" \
     -H "X-TIMEOUT: 5" \
     -H "X-SESSION-ID: test-session-3" \
     "$PROXY_URL" | jq .
echo

echo "4. Testing POST request with JSON body..."
curl -s -X POST \
     -H "X-URL: https://httpbin.org/post" \
     -H "X-IDENTIFIER: safari" \
     -H "X-SESSION-ID: test-session-4" \
     -H "Content-Type: application/json" \
     -d '{"test": "data", "proxy": "cycletls"}' \
     "$PROXY_URL" | jq .
echo

echo "5. Testing request with custom headers..."
curl -s -H "X-URL: https://httpbin.org/headers" \
     -H "X-IDENTIFIER: edge" \
     -H "X-SESSION-ID: test-session-5" \
     -H "Custom-Header: test-value" \
     -H "Authorization: Bearer token123" \
     "$PROXY_URL" | jq .
echo

echo "6. Testing session reuse (same session ID)..."
curl -s -H "X-URL: https://httpbin.org/get?request=1" \
     -H "X-IDENTIFIER: chrome" \
     -H "X-SESSION-ID: reuse-session" \
     "$PROXY_URL" | jq . > /dev/null
echo "   First request sent..."

curl -s -H "X-URL: https://httpbin.org/get?request=2" \
     -H "X-IDENTIFIER: chrome" \
     -H "X-SESSION-ID: reuse-session" \
     "$PROXY_URL" | jq .
echo

echo "7. Testing error handling (invalid profile)..."
curl -s -H "X-URL: https://httpbin.org/get" \
     -H "X-IDENTIFIER: invalid-profile" \
     "$PROXY_URL"
echo
echo

echo "8. Testing error handling (missing X-URL)..."
curl -s -H "X-IDENTIFIER: chrome" \
     "$PROXY_URL"
echo
echo

echo "✅ All tests completed!"
echo "Check the server logs for detailed request information."