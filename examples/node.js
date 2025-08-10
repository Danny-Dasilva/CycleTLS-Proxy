#!/usr/bin/env node

/**
 * CycleTLS-Proxy Node.js Client Examples
 * 
 * This module provides comprehensive examples and a Node.js client library
 * for interacting with the CycleTLS-Proxy server. It demonstrates various
 * use cases including different browser profiles, session management,
 * authentication, and error handling.
 * 
 * Requirements:
 *   npm install axios
 * 
 * Usage:
 *   node examples/node.js
 * 
 * Or import as a library:
 *   const { CycleTLSClient, BrowserProfile } = require('./examples/node.js');
 */

const axios = require('axios');
const { performance } = require('perf_hooks');
const { v4: uuidv4 } = require('crypto');

// Browser profile enumeration
const BrowserProfile = {
    CHROME: 'chrome',
    CHROME_WINDOWS: 'chrome_windows',
    FIREFOX: 'firefox',
    FIREFOX_WINDOWS: 'firefox_windows',
    SAFARI: 'safari',
    SAFARI_IOS: 'safari_ios',
    EDGE: 'edge',
    OKHTTP: 'okhttp',
    CHROME_LEGACY_TLS12: 'chrome_legacy_tls12'
};

// Custom error classes
class CycleTLSError extends Error {
    constructor(message) {
        super(message);
        this.name = 'CycleTLSError';
    }
}

class CycleTLSTimeoutError extends CycleTLSError {
    constructor(message) {
        super(message);
        this.name = 'CycleTLSTimeoutError';
    }
}

class CycleTLSInvalidProfileError extends CycleTLSError {
    constructor(message) {
        super(message);
        this.name = 'CycleTLSInvalidProfileError';
    }
}

/**
 * Node.js client for CycleTLS-Proxy server.
 * 
 * This client provides a convenient interface for making HTTP requests
 * through the CycleTLS-Proxy with various browser fingerprints.
 * 
 * @example
 * const client = new CycleTLSClient();
 * const response = await client.get("https://httpbin.org/json", { profile: BrowserProfile.CHROME });
 * console.log(response.data);
 * 
 * @example
 * // Using sessions
 * const session = client.createSession("my-session");
 * await session.post("https://httpbin.org/post", { json: { key: "value" } });
 */
class CycleTLSClient {
    /**
     * Initialize the CycleTLS client.
     * 
     * @param {Object} options - Configuration options
     * @param {string} options.proxyUrl - URL of the CycleTLS-Proxy server
     * @param {number} options.defaultTimeout - Default timeout for requests in seconds
     * @param {number} options.maxRetries - Maximum number of retry attempts
     */
    constructor(options = {}) {
        this.proxyUrl = (options.proxyUrl || 'http://localhost:8080').replace(/\/$/, '');
        this.defaultTimeout = options.defaultTimeout || 30;
        
        // Configure axios instance with retries
        this.axios = axios.create({
            timeout: (this.defaultTimeout + 5) * 1000,
            maxRedirects: 0, // Handle redirects at proxy level
        });
        
        // Add retry interceptor
        this._setupRetryInterceptor(options.maxRetries || 3);
    }
    
    /**
     * Setup automatic retry logic for failed requests.
     * @private
     */
    _setupRetryInterceptor(maxRetries) {
        this.axios.interceptors.response.use(
            response => response,
            async error => {
                const config = error.config;
                
                if (!config._retryCount) config._retryCount = 0;
                
                if (config._retryCount >= maxRetries) {
                    return Promise.reject(error);
                }
                
                // Retry on network errors and 5xx status codes
                if (error.code === 'ECONNRESET' || 
                    error.code === 'ENOTFOUND' ||
                    (error.response && error.response.status >= 500)) {
                    
                    config._retryCount++;
                    
                    // Exponential backoff
                    const delay = Math.pow(2, config._retryCount) * 1000;
                    await new Promise(resolve => setTimeout(resolve, delay));
                    
                    return this.axios(config);
                }
                
                return Promise.reject(error);
            }
        );
    }
    
    /**
     * Create headers for the proxy request.
     * @private
     */
    _makeHeaders(config, extraHeaders = {}) {
        const headers = {
            'X-URL': config.url,
            'X-IDENTIFIER': config.profile || BrowserProfile.CHROME,
        };
        
        if (config.sessionId) {
            headers['X-SESSION-ID'] = config.sessionId;
        }
        
        if (config.upstreamProxy) {
            headers['X-PROXY'] = config.upstreamProxy;
        }
        
        if (config.timeout && config.timeout !== this.defaultTimeout) {
            headers['X-TIMEOUT'] = config.timeout.toString();
        }
        
        return { ...headers, ...extraHeaders };
    }
    
    /**
     * Handle and validate proxy response.
     * @private
     */
    _handleError(error) {
        if (error.response) {
            const { status, data } = error.response;
            const message = typeof data === 'string' ? data : data.message || 'Unknown error';
            
            switch (status) {
                case 400:
                    if (message.includes('Invalid identifier')) {
                        throw new CycleTLSInvalidProfileError(`Invalid browser profile: ${message}`);
                    } else if (message.toLowerCase().includes('timeout')) {
                        throw new CycleTLSTimeoutError(`Request timeout: ${message}`);
                    } else {
                        throw new CycleTLSError(`Bad request: ${message}`);
                    }
                case 502:
                    throw new CycleTLSError(`Upstream request failed: ${message}`);
                default:
                    throw new CycleTLSError(`HTTP ${status}: ${message}`);
            }
        } else if (error.code === 'ECONNABORTED') {
            throw new CycleTLSTimeoutError('Request timed out');
        } else if (error.code === 'ECONNREFUSED') {
            throw new CycleTLSError('Connection refused - is the CycleTLS-Proxy server running?');
        } else {
            throw new CycleTLSError(`Network error: ${error.message}`);
        }
    }
    
    /**
     * Make a request through the CycleTLS-Proxy.
     * 
     * @param {string} method - HTTP method (GET, POST, PUT, etc.)
     * @param {string} url - Target URL to request
     * @param {Object} options - Request options
     * @param {string} options.profile - Browser profile to use for TLS fingerprinting
     * @param {string} options.sessionId - Optional session ID for connection reuse
     * @param {string} options.upstreamProxy - Optional upstream proxy URL
     * @param {number} options.timeout - Request timeout in seconds
     * @param {Object} options.headers - Additional headers to send
     * @param {*} options.data - Request body data
     * @param {Object} options.json - JSON data to send (sets appropriate headers)
     * @param {Object} options.params - URL parameters
     * 
     * @returns {Promise<Object>} Axios response object
     * @throws {CycleTLSError} On various proxy-related errors
     */
    async request(method, url, options = {}) {
        const config = {
            url,
            profile: options.profile,
            sessionId: options.sessionId,
            upstreamProxy: options.upstreamProxy,
            timeout: options.timeout || this.defaultTimeout
        };
        
        const proxyHeaders = this._makeHeaders(config, options.headers);
        
        // Prepare request data
        let requestData = options.data;
        if (options.json) {
            requestData = options.json;
            proxyHeaders['Content-Type'] = 'application/json';
        }
        
        // Prepare URL parameters
        let proxyUrl = this.proxyUrl;
        if (options.params) {
            // Note: URL params should be added to the X-URL, not the proxy URL
            const urlObj = new URL(url);
            Object.entries(options.params).forEach(([key, value]) => {
                urlObj.searchParams.append(key, value);
            });
            proxyHeaders['X-URL'] = urlObj.toString();
        }
        
        try {
            const response = await this.axios.request({
                method,
                url: proxyUrl,
                headers: proxyHeaders,
                data: requestData,
                timeout: (config.timeout + 5) * 1000, // Add buffer for proxy overhead
                validateStatus: () => true // Accept all status codes, let proxy handle them
            });
            
            return response;
        } catch (error) {
            this._handleError(error);
        }
    }
    
    /**
     * Make a GET request.
     * @param {string} url - Target URL
     * @param {Object} options - Request options
     * @returns {Promise<Object>} Response object
     */
    async get(url, options = {}) {
        return this.request('GET', url, options);
    }
    
    /**
     * Make a POST request.
     * @param {string} url - Target URL
     * @param {Object} options - Request options
     * @returns {Promise<Object>} Response object
     */
    async post(url, options = {}) {
        return this.request('POST', url, options);
    }
    
    /**
     * Make a PUT request.
     * @param {string} url - Target URL
     * @param {Object} options - Request options
     * @returns {Promise<Object>} Response object
     */
    async put(url, options = {}) {
        return this.request('PUT', url, options);
    }
    
    /**
     * Make a PATCH request.
     * @param {string} url - Target URL
     * @param {Object} options - Request options
     * @returns {Promise<Object>} Response object
     */
    async patch(url, options = {}) {
        return this.request('PATCH', url, options);
    }
    
    /**
     * Make a DELETE request.
     * @param {string} url - Target URL
     * @param {Object} options - Request options
     * @returns {Promise<Object>} Response object
     */
    async delete(url, options = {}) {
        return this.request('DELETE', url, options);
    }
    
    /**
     * Make a HEAD request.
     * @param {string} url - Target URL
     * @param {Object} options - Request options
     * @returns {Promise<Object>} Response object
     */
    async head(url, options = {}) {
        return this.request('HEAD', url, options);
    }
    
    /**
     * Create a session for persistent connections.
     * 
     * @param {string} sessionId - Session ID, auto-generated if not provided
     * @returns {SessionContext} Session context object
     */
    createSession(sessionId = null) {
        if (!sessionId) {
            sessionId = `nodejs-session-${uuidv4().substring(0, 8)}`;
        }
        
        return new SessionContext(this, sessionId);
    }
    
    /**
     * Check the health status of the CycleTLS-Proxy server.
     * 
     * @returns {Promise<Object>} Health information
     * @throws {CycleTLSError} If health check fails
     */
    async healthCheck() {
        try {
            const response = await this.axios.get(`${this.proxyUrl}/health`, { timeout: 5000 });
            return response.data;
        } catch (error) {
            throw new CycleTLSError(`Health check failed: ${error.message}`);
        }
    }
    
    /**
     * Get list of available browser profiles from server.
     * 
     * @returns {Promise<string[]>} List of available profile identifiers
     */
    async getAvailableProfiles() {
        try {
            const health = await this.healthCheck();
            // Try to get profiles from health endpoint or use fallback
            return Object.values(BrowserProfile);
        } catch (error) {
            // Try to extract profiles from error message
            try {
                await this.get('https://httpbin.org/get', { profile: 'invalid-profile-test' });
            } catch (err) {
                if (err instanceof CycleTLSInvalidProfileError) {
                    const match = err.message.match(/Available profiles: ([^"]+)/);
                    if (match) {
                        return match[1].split(',').map(p => p.trim());
                    }
                }
            }
            
            // Fallback to known profiles
            return Object.values(BrowserProfile);
        }
    }
}

/**
 * Session context for persistent CycleTLS connections.
 */
class SessionContext {
    constructor(client, sessionId) {
        this.client = client;
        this.sessionId = sessionId;
    }
    
    /**
     * Make a request using this session.
     */
    async request(method, url, options = {}) {
        return this.client.request(method, url, { ...options, sessionId: this.sessionId });
    }
    
    async get(url, options = {}) {
        return this.request('GET', url, options);
    }
    
    async post(url, options = {}) {
        return this.request('POST', url, options);
    }
    
    async put(url, options = {}) {
        return this.request('PUT', url, options);
    }
    
    async patch(url, options = {}) {
        return this.request('PATCH', url, options);
    }
    
    async delete(url, options = {}) {
        return this.request('DELETE', url, options);
    }
    
    async head(url, options = {}) {
        return this.request('HEAD', url, options);
    }
}

// Example functions
async function runBasicExamples() {
    console.log('🚀 CycleTLS-Proxy Node.js Client Examples');
    console.log('='.repeat(50));
    
    const client = new CycleTLSClient();
    
    // Health check
    console.log('\n📊 Health Check');
    try {
        const health = await client.healthCheck();
        console.log(`✓ Server status: ${health.status}`);
        console.log(`✓ Available profiles: ${health.proxy.profiles_available}`);
        console.log(`✓ Active sessions: ${health.proxy.active_sessions}`);
    } catch (error) {
        console.log(`✗ Health check failed: ${error.message}`);
        return;
    }
    
    console.log('\n🌐 Basic GET Requests with Different Profiles');
    
    // Test different browser profiles
    const profilesToTest = [
        [BrowserProfile.CHROME, 'Chrome Linux'],
        [BrowserProfile.FIREFOX, 'Firefox Linux'],
        [BrowserProfile.SAFARI_IOS, 'Safari iOS'],
        [BrowserProfile.EDGE, 'Microsoft Edge'],
    ];
    
    for (const [profile, description] of profilesToTest) {
        try {
            const response = await client.get('https://httpbin.org/user-agent', { profile });
            const userAgent = response.data['user-agent'];
            console.log(`✓ ${description}: ${userAgent}`);
        } catch (error) {
            console.log(`✗ ${description}: ${error.message}`);
        }
    }
    
    console.log('\n📝 HTTP Methods');
    
    // POST request with JSON
    try {
        const data = {
            username: 'testuser',
            email: 'test@example.com',
            active: true,
            metadata: { source: 'nodejs-example' }
        };
        const response = await client.post('https://httpbin.org/post', {
            profile: BrowserProfile.CHROME,
            json: data
        });
        console.log(`✓ POST with JSON: ${response.data.json.username}`);
    } catch (error) {
        console.log(`✗ POST request failed: ${error.message}`);
    }
    
    // PUT request
    try {
        const response = await client.put('https://httpbin.org/put', {
            profile: BrowserProfile.FIREFOX,
            data: 'Updated content',
            headers: { 'Content-Type': 'text/plain' }
        });
        console.log(`✓ PUT request: Status ${response.status}`);
    } catch (error) {
        console.log(`✗ PUT request failed: ${error.message}`);
    }
    
    // DELETE request
    try {
        const response = await client.delete('https://httpbin.org/delete', {
            profile: BrowserProfile.SAFARI
        });
        console.log(`✓ DELETE request: Status ${response.status}`);
    } catch (error) {
        console.log(`✗ DELETE request failed: ${error.message}`);
    }
}

async function runSessionExamples() {
    console.log('\n🔄 Session Management');
    
    const client = new CycleTLSClient();
    const session = client.createSession('demo-session');
    
    try {
        // Set a cookie
        await session.get('https://httpbin.org/cookies/set/session_token/abc123');
        
        // Verify cookie persistence
        const response = await session.get('https://httpbin.org/cookies');
        const cookies = response.data.cookies || {};
        if (cookies.session_token) {
            console.log(`✓ Session cookie persisted: ${cookies.session_token}`);
        } else {
            console.log('✗ Session cookie not found');
        }
        
        // Add another cookie
        await session.get('https://httpbin.org/cookies/set/user_id/12345');
        
        // Check both cookies
        const response2 = await session.get('https://httpbin.org/cookies');
        const cookies2 = response2.data.cookies || {};
        console.log(`✓ Session has ${Object.keys(cookies2).length} cookies: ${Object.keys(cookies2).join(', ')}`);
        
    } catch (error) {
        console.log(`✗ Session example failed: ${error.message}`);
    }
}

async function runAuthenticationExamples() {
    console.log('\n🔐 Authentication Examples');
    
    const client = new CycleTLSClient();
    
    // Basic authentication
    try {
        const credentials = Buffer.from('testuser:testpass').toString('base64');
        const response = await client.get('https://httpbin.org/basic-auth/testuser/testpass', {
            headers: { 'Authorization': `Basic ${credentials}` },
            profile: BrowserProfile.CHROME
        });
        console.log(`✓ Basic auth: ${response.data.authenticated}`);
    } catch (error) {
        console.log(`✗ Basic auth failed: ${error.message}`);
    }
    
    // Bearer token authentication
    try {
        const response = await client.get('https://httpbin.org/bearer', {
            headers: { 'Authorization': 'Bearer test-token-12345' },
            profile: BrowserProfile.FIREFOX
        });
        console.log(`✓ Bearer token: ${response.data.authenticated}`);
    } catch (error) {
        console.log(`✗ Bearer token failed: ${error.message}`);
    }
}

async function runAdvancedExamples() {
    console.log('\n🚀 Advanced Features');
    
    const client = new CycleTLSClient();
    
    // Custom headers
    try {
        const headers = {
            'X-API-Key': 'secret-key-123',
            'X-Client-Version': '1.0.0',
            'Accept': 'application/json',
            'User-Agent': 'This-Should-Be-Overridden/1.0'
        };
        const response = await client.get('https://httpbin.org/headers', {
            headers,
            profile: BrowserProfile.SAFARI_IOS
        });
        const resultHeaders = response.data.headers;
        console.log(`✓ Custom headers forwarded: ${Object.keys(resultHeaders).length} headers`);
        
        // Check if User-Agent was correctly overridden by profile
        if (resultHeaders['User-Agent'] && resultHeaders['User-Agent'].includes('iPhone')) {
            console.log('✓ Profile User-Agent correctly used');
        } else {
            console.log('✗ Custom User-Agent incorrectly used');
        }
        
    } catch (error) {
        console.log(`✗ Custom headers failed: ${error.message}`);
    }
    
    // Timeout configuration
    try {
        const startTime = performance.now();
        const response = await client.get('https://httpbin.org/delay/1', {
            timeout: 5,
            profile: BrowserProfile.CHROME
        });
        const duration = (performance.now() - startTime) / 1000;
        console.log(`✓ Timeout test passed: ${duration.toFixed(2)}s`);
    } catch (error) {
        console.log(`✗ Timeout test failed: ${error.message}`);
    }
    
    // URL parameters
    try {
        const response = await client.get('https://httpbin.org/get', {
            params: {
                page: 1,
                limit: 10,
                search: 'nodejs example'
            },
            profile: BrowserProfile.EDGE
        });
        const args = response.data.args;
        console.log(`✓ URL parameters: ${Object.keys(args).length} params sent`);
    } catch (error) {
        console.log(`✗ URL parameters failed: ${error.message}`);
    }
    
    // Large JSON payload
    try {
        const largeData = {
            items: Array.from({ length: 100 }, (_, i) => ({
                id: i,
                name: `Item ${i}`,
                active: true
            })),
            metadata: {
                total: 100,
                generated: Date.now(),
                client: 'nodejs-example'
            }
        };
        const response = await client.post('https://httpbin.org/post', {
            json: largeData,
            profile: BrowserProfile.EDGE
        });
        const itemsCount = response.data.json.items.length;
        console.log(`✓ Large JSON payload: ${itemsCount} items sent`);
    } catch (error) {
        console.log(`✗ Large JSON payload failed: ${error.message}`);
    }
}

async function runErrorHandlingExamples() {
    console.log('\n⚠️  Error Handling');
    
    const client = new CycleTLSClient();
    
    // Invalid profile
    try {
        await client.get('https://httpbin.org/get', { profile: 'invalid-profile' });
        console.log('✗ Invalid profile should have failed');
    } catch (error) {
        if (error instanceof CycleTLSInvalidProfileError) {
            console.log('✓ Invalid profile error handled correctly');
        } else {
            console.log(`✗ Unexpected error for invalid profile: ${error.message}`);
        }
    }
    
    // Empty URL
    try {
        await client.get('', { profile: BrowserProfile.CHROME });
        console.log('✗ Empty URL should have failed');
    } catch (error) {
        console.log('✓ Empty URL error handled correctly');
    }
    
    // Invalid URL
    try {
        await client.get('not-a-url', { profile: BrowserProfile.CHROME });
        console.log('✗ Invalid URL should have failed');
    } catch (error) {
        console.log('✓ Invalid URL error handled correctly');
    }
    
    // Timeout error
    try {
        await client.get('https://httpbin.org/delay/5', {
            timeout: 1,
            profile: BrowserProfile.CHROME
        });
        console.log('✗ Timeout should have occurred');
    } catch (error) {
        if (error instanceof CycleTLSTimeoutError) {
            console.log('✓ Timeout error handled correctly');
        } else {
            console.log(`✗ Unexpected timeout error: ${error.message}`);
        }
    }
}

async function runConcurrentExamples() {
    console.log('\n🔄 Concurrent Requests');
    
    const client = new CycleTLSClient();
    
    const makeRequest = async (requestId) => {
        try {
            const response = await client.get(`https://httpbin.org/get?request_id=${requestId}`, {
                profile: BrowserProfile.CHROME,
                sessionId: `concurrent-${requestId}`
            });
            return { requestId, status: response.status, size: JSON.stringify(response.data).length };
        } catch (error) {
            return { requestId, status: 'ERROR', error: error.message };
        }
    };
    
    // Run 10 concurrent requests
    const startTime = performance.now();
    const promises = Array.from({ length: 10 }, (_, i) => makeRequest(i));
    const results = await Promise.all(promises);
    const duration = (performance.now() - startTime) / 1000;
    
    const successful = results.filter(r => r.status === 200).length;
    console.log(`✓ Completed ${successful}/10 concurrent requests in ${duration.toFixed(2)}s`);
    
    results.forEach(result => {
        if (result.status === 200) {
            console.log(`  Request ${result.requestId}: HTTP ${result.status}, ${result.size} bytes`);
        } else {
            console.log(`  Request ${result.requestId}: ${result.status} - ${result.error || 'Unknown error'}`);
        }
    });
}

async function runRealWorldExamples() {
    console.log('\n🌍 Real-World Examples');
    
    const client = new CycleTLSClient();
    
    // Simulate login flow
    console.log('\n🔐 Simulated Login Flow');
    try {
        const session = client.createSession('login-demo');
        
        // Get login page (simulate)
        await session.get('https://httpbin.org/get', {
            profile: BrowserProfile.CHROME
        });
        console.log('✓ Step 1: Retrieved login page');
        
        // Submit login credentials
        const loginData = {
            username: 'demo_user',
            password: 'secure_password',
            remember_me: true
        };
        await session.post('https://httpbin.org/post', {
            json: loginData,
            profile: BrowserProfile.CHROME
        });
        console.log('✓ Step 2: Submitted login credentials');
        
        // Access protected resource
        await session.get('https://httpbin.org/get?protected=true', {
            profile: BrowserProfile.CHROME,
            headers: { 'Authorization': 'Bearer fake-jwt-token' }
        });
        console.log('✓ Step 3: Accessed protected resource');
        
    } catch (error) {
        console.log(`✗ Login flow failed: ${error.message}`);
    }
    
    // API interaction example
    console.log('\n📡 API Interaction Example');
    try {
        // Create resource
        const createData = {
            name: 'Test Resource',
            description: 'Created via CycleTLS-Proxy',
            active: true,
            tags: ['test', 'example', 'nodejs']
        };
        await client.post('https://httpbin.org/post', {
            json: createData,
            profile: BrowserProfile.CHROME,
            headers: {
                'X-API-Key': 'demo-api-key-123'
            }
        });
        console.log('✓ Created resource via API');
        
        // Update resource
        const updateData = { name: 'Updated Test Resource' };
        await client.patch('https://httpbin.org/patch', {
            json: updateData,
            profile: BrowserProfile.CHROME,
            headers: {
                'X-API-Key': 'demo-api-key-123'
            }
        });
        console.log('✓ Updated resource via API');
        
        // List resources
        await client.get('https://httpbin.org/get', {
            params: { page: 1, limit: 10 },
            profile: BrowserProfile.CHROME,
            headers: { 'X-API-Key': 'demo-api-key-123' }
        });
        console.log('✓ Retrieved resource list via API');
        
    } catch (error) {
        console.log(`✗ API interaction failed: ${error.message}`);
    }
}

async function runPerformanceExamples() {
    console.log('\n⚡ Performance Examples');
    
    const client = new CycleTLSClient();
    
    // Measure single request latency
    try {
        const startTime = performance.now();
        await client.get('https://httpbin.org/get', { profile: BrowserProfile.CHROME });
        const latency = performance.now() - startTime;
        console.log(`✓ Single request latency: ${latency.toFixed(2)}ms`);
    } catch (error) {
        console.log(`✗ Latency test failed: ${error.message}`);
    }
    
    // Measure throughput with sequential requests
    try {
        const requestCount = 20;
        const startTime = performance.now();
        
        for (let i = 0; i < requestCount; i++) {
            await client.get(`https://httpbin.org/get?seq=${i}`, {
                profile: BrowserProfile.CHROME,
                sessionId: 'throughput-test'
            });
        }
        
        const duration = (performance.now() - startTime) / 1000;
        const throughput = requestCount / duration;
        console.log(`✓ Sequential throughput: ${throughput.toFixed(2)} req/s (${requestCount} requests in ${duration.toFixed(2)}s)`);
    } catch (error) {
        console.log(`✗ Throughput test failed: ${error.message}`);
    }
    
    // Test different response sizes
    const sizes = [100, 1000, 10000];
    for (const size of sizes) {
        try {
            const startTime = performance.now();
            const response = await client.get(`https://httpbin.org/drip?duration=0&numbytes=${size}`, {
                profile: BrowserProfile.CHROME
            });
            const duration = performance.now() - startTime;
            const actualSize = response.data.length || 0;
            console.log(`✓ ${size} bytes response: ${duration.toFixed(2)}ms (actual: ${actualSize} bytes)`);
        } catch (error) {
            console.log(`✗ ${size} bytes test failed: ${error.message}`);
        }
    }
}

// Main execution function
async function main() {
    console.log('CycleTLS-Proxy Node.js Client - Comprehensive Examples');
    console.log('='.repeat(60));
    
    try {
        // Check if axios is available
        if (!axios) {
            throw new Error('axios is required but not installed. Run: npm install axios');
        }
        
        // Basic examples
        await runBasicExamples();
        
        // Session management
        await runSessionExamples();
        
        // Authentication
        await runAuthenticationExamples();
        
        // Advanced features
        await runAdvancedExamples();
        
        // Error handling
        await runErrorHandlingExamples();
        
        // Concurrent requests
        await runConcurrentExamples();
        
        // Real-world examples
        await runRealWorldExamples();
        
        // Performance examples
        await runPerformanceExamples();
        
        console.log('\n🎉 All Examples Completed Successfully!');
        console.log('\nTo use this as a library:');
        console.log('```javascript');
        console.log("const { CycleTLSClient, BrowserProfile } = require('./examples/node.js');");
        console.log('');
        console.log('const client = new CycleTLSClient();');
        console.log("const response = await client.get('https://api.example.com', { profile: BrowserProfile.CHROME });");
        console.log('console.log(response.data);');
        console.log('```');
        
    } catch (error) {
        if (error.code === 'MODULE_NOT_FOUND' && error.message.includes('axios')) {
            console.error('\n❌ Missing dependency: axios');
            console.error('Please install it with: npm install axios');
            process.exit(1);
        } else {
            console.error(`\n❌ Examples failed: ${error.message}`);
            process.exit(1);
        }
    }
}

// Export classes for use as a library
module.exports = {
    CycleTLSClient,
    SessionContext,
    BrowserProfile,
    CycleTLSError,
    CycleTLSTimeoutError,
    CycleTLSInvalidProfileError
};

// Run examples if this file is executed directly
if (require.main === module) {
    main().catch(error => {
        console.error(`\n❌ Fatal error: ${error.message}`);
        process.exit(1);
    });
}