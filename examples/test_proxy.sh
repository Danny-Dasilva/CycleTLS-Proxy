#!/bin/bash

# CycleTLS Proxy Test Script
# This script demonstrates how to use the CycleTLS proxy

echo "🧪 Testing CycleTLS Proxy"
echo "========================="

PROXY_URL="http://localhost:8080"

# Test 1: Basic GET request with Chrome fingerprint
echo ""
echo "📄 Test 1: Basic GET with Chrome fingerprint"
echo "curl -H 'X-URL: https://httpbin.org/json' -H 'X-IDENTIFIER: chrome' $PROXY_URL"
curl -H "X-URL: https://httpbin.org/json" -H "X-IDENTIFIER: chrome" $PROXY_URL | jq '.'

# Test 2: GET request with Firefox fingerprint
echo ""
echo "🦊 Test 2: GET with Firefox fingerprint"
echo "curl -H 'X-URL: https://httpbin.org/user-agent' -H 'X-IDENTIFIER: firefox' $PROXY_URL"
curl -H "X-URL: https://httpbin.org/user-agent" -H "X-IDENTIFIER: firefox" $PROXY_URL | jq '.'

# Test 3: POST request with session
echo ""
echo "📤 Test 3: POST with session ID"
echo "curl -X POST -H 'X-URL: https://httpbin.org/post' -H 'X-IDENTIFIER: chrome' -H 'X-SESSION-ID: test-session' -d '{\"test\": \"data\"}' $PROXY_URL"
curl -X POST \
     -H "X-URL: https://httpbin.org/post" \
     -H "X-IDENTIFIER: chrome" \
     -H "X-SESSION-ID: test-session" \
     -H "Content-Type: application/json" \
     -d '{"test": "data"}' \
     $PROXY_URL | jq '.json'

# Test 4: Request with custom headers
echo ""
echo "🎯 Test 4: Request with custom headers"
echo "curl -H 'X-URL: https://httpbin.org/headers' -H 'X-IDENTIFIER: safari' -H 'Custom-Header: test-value' $PROXY_URL"
curl -H "X-URL: https://httpbin.org/headers" \
     -H "X-IDENTIFIER: safari" \
     -H "Custom-Header: test-value" \
     $PROXY_URL | jq '.headers'

echo ""
echo "✅ All tests completed!"