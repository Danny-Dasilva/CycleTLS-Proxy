#!/usr/bin/env python3
"""
CycleTLS-Proxy Python Client Examples

This module provides comprehensive examples and a Python client library
for interacting with the CycleTLS-Proxy server. It demonstrates various
use cases including different browser profiles, session management,
authentication, and error handling.

Requirements:
    pip install requests

Usage:
    python examples/python.py

Or import as a library:
    from examples.python import CycleTLSClient
"""

import json
import time
import uuid
import asyncio
import concurrent.futures
from typing import Dict, Optional, Any, List, Union
from dataclasses import dataclass, asdict
from enum import Enum

try:
    import requests
    from requests.adapters import HTTPAdapter
    from urllib3.util.retry import Retry
except ImportError as e:
    print("Error: requests library is required")
    print("Install with: pip install requests")
    raise e

try:
    import aiohttp
    AIOHTTP_AVAILABLE = True
except ImportError:
    AIOHTTP_AVAILABLE = False
    print("Warning: aiohttp not available for async examples")
    print("Install with: pip install aiohttp for async support")


class BrowserProfile(Enum):
    """Available browser profiles for TLS fingerprinting."""
    CHROME = "chrome"
    CHROME_WINDOWS = "chrome_windows"
    FIREFOX = "firefox"
    FIREFOX_WINDOWS = "firefox_windows"
    SAFARI = "safari"
    SAFARI_IOS = "safari_ios"
    EDGE = "edge"
    OKHTTP = "okhttp"
    CHROME_LEGACY_TLS12 = "chrome_legacy_tls12"


@dataclass
class ProxyConfig:
    """Configuration for CycleTLS-Proxy requests."""
    url: str
    profile: Union[str, BrowserProfile] = BrowserProfile.CHROME
    session_id: Optional[str] = None
    upstream_proxy: Optional[str] = None
    timeout: int = 30
    
    def __post_init__(self):
        """Convert BrowserProfile enum to string."""
        if isinstance(self.profile, BrowserProfile):
            self.profile = self.profile.value


class CycleTLSError(Exception):
    """Base exception for CycleTLS-Proxy errors."""
    pass


class CycleTLSTimeoutError(CycleTLSError):
    """Raised when a request times out."""
    pass


class CycleTLSInvalidProfileError(CycleTLSError):
    """Raised when an invalid browser profile is specified."""
    pass


class CycleTLSClient:
    """
    Python client for CycleTLS-Proxy server.
    
    This client provides a convenient interface for making HTTP requests
    through the CycleTLS-Proxy with various browser fingerprints.
    
    Examples:
        >>> client = CycleTLSClient()
        >>> response = client.get("https://httpbin.org/json", profile=BrowserProfile.CHROME)
        >>> print(response.json())
        
        >>> # Using sessions
        >>> with client.session("my-session") as session:
        ...     session.post("https://httpbin.org/post", json={"key": "value"})
    """
    
    def __init__(self, proxy_url: str = "http://localhost:8080", 
                 default_timeout: int = 30, max_retries: int = 3):
        """
        Initialize the CycleTLS client.
        
        Args:
            proxy_url: URL of the CycleTLS-Proxy server
            default_timeout: Default timeout for requests in seconds
            max_retries: Maximum number of retry attempts
        """
        self.proxy_url = proxy_url.rstrip('/')
        self.default_timeout = default_timeout
        
        # Configure requests session with retries
        self.session = requests.Session()
        retry_strategy = Retry(
            total=max_retries,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
    
    def _make_headers(self, config: ProxyConfig, extra_headers: Optional[Dict[str, str]] = None) -> Dict[str, str]:
        """Create headers for the proxy request."""
        headers = {
            "X-URL": config.url,
            "X-IDENTIFIER": config.profile,
        }
        
        if config.session_id:
            headers["X-SESSION-ID"] = config.session_id
        
        if config.upstream_proxy:
            headers["X-PROXY"] = config.upstream_proxy
        
        if config.timeout != self.default_timeout:
            headers["X-TIMEOUT"] = str(config.timeout)
        
        if extra_headers:
            headers.update(extra_headers)
        
        return headers
    
    def _handle_response(self, response: requests.Response) -> requests.Response:
        """Handle and validate proxy response."""
        try:
            response.raise_for_status()
        except requests.exceptions.HTTPError as e:
            if response.status_code == 400:
                error_text = response.text
                if "Invalid identifier" in error_text:
                    raise CycleTLSInvalidProfileError(f"Invalid browser profile: {error_text}")
                elif "timeout" in error_text.lower():
                    raise CycleTLSTimeoutError(f"Request timeout: {error_text}")
                else:
                    raise CycleTLSError(f"Bad request: {error_text}")
            elif response.status_code == 502:
                raise CycleTLSError(f"Upstream request failed: {response.text}")
            else:
                raise CycleTLSError(f"HTTP {response.status_code}: {response.text}")
        
        return response
    
    def request(self, method: str, url: str, 
                profile: Union[str, BrowserProfile] = BrowserProfile.CHROME,
                session_id: Optional[str] = None,
                upstream_proxy: Optional[str] = None,
                timeout: Optional[int] = None,
                headers: Optional[Dict[str, str]] = None,
                **kwargs) -> requests.Response:
        """
        Make a request through the CycleTLS-Proxy.
        
        Args:
            method: HTTP method (GET, POST, PUT, etc.)
            url: Target URL to request
            profile: Browser profile to use for TLS fingerprinting
            session_id: Optional session ID for connection reuse
            upstream_proxy: Optional upstream proxy URL
            timeout: Request timeout in seconds
            headers: Additional headers to send
            **kwargs: Additional arguments passed to requests
            
        Returns:
            requests.Response object
            
        Raises:
            CycleTLSError: On various proxy-related errors
        """
        config = ProxyConfig(
            url=url,
            profile=profile,
            session_id=session_id,
            upstream_proxy=upstream_proxy,
            timeout=timeout or self.default_timeout
        )
        
        proxy_headers = self._make_headers(config, headers)
        
        try:
            response = self.session.request(
                method=method,
                url=self.proxy_url,
                headers=proxy_headers,
                timeout=config.timeout + 5,  # Add buffer for proxy overhead
                **kwargs
            )
            return self._handle_response(response)
        
        except requests.exceptions.Timeout:
            raise CycleTLSTimeoutError(f"Request timed out after {config.timeout}s")
        except requests.exceptions.ConnectionError as e:
            raise CycleTLSError(f"Connection error: {e}")
    
    def get(self, url: str, **kwargs) -> requests.Response:
        """Make a GET request."""
        return self.request("GET", url, **kwargs)
    
    def post(self, url: str, **kwargs) -> requests.Response:
        """Make a POST request."""
        return self.request("POST", url, **kwargs)
    
    def put(self, url: str, **kwargs) -> requests.Response:
        """Make a PUT request."""
        return self.request("PUT", url, **kwargs)
    
    def patch(self, url: str, **kwargs) -> requests.Response:
        """Make a PATCH request."""
        return self.request("PATCH", url, **kwargs)
    
    def delete(self, url: str, **kwargs) -> requests.Response:
        """Make a DELETE request."""
        return self.request("DELETE", url, **kwargs)
    
    def head(self, url: str, **kwargs) -> requests.Response:
        """Make a HEAD request."""
        return self.request("HEAD", url, **kwargs)
    
    def session(self, session_id: Optional[str] = None) -> 'SessionContext':
        """
        Create a session context for persistent connections.
        
        Args:
            session_id: Session ID, auto-generated if None
            
        Returns:
            SessionContext object for use in 'with' statements
        """
        if session_id is None:
            session_id = f"python-session-{uuid.uuid4().hex[:8]}"
        
        return SessionContext(self, session_id)
    
    def health_check(self) -> Dict[str, Any]:
        """
        Check the health status of the CycleTLS-Proxy server.
        
        Returns:
            Dictionary containing server health information
        """
        try:
            response = self.session.get(f"{self.proxy_url}/health", timeout=5)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            raise CycleTLSError(f"Health check failed: {e}")
    
    def get_available_profiles(self) -> List[str]:
        """
        Get list of available browser profiles from server.
        
        Returns:
            List of available profile identifiers
        """
        health = self.health_check()
        # Parse available profiles from error message by trying invalid profile
        try:
            self.get("https://httpbin.org/get", profile="invalid-profile-test")
        except CycleTLSInvalidProfileError as e:
            import re
            match = re.search(r'Available profiles: ([^"]+)', str(e))
            if match:
                return [p.strip() for p in match.group(1).split(',')]
        
        # Fallback to known profiles
        return [profile.value for profile in BrowserProfile]


class SessionContext:
    """Context manager for persistent CycleTLS sessions."""
    
    def __init__(self, client: CycleTLSClient, session_id: str):
        self.client = client
        self.session_id = session_id
    
    def __enter__(self):
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        # Sessions are automatically cleaned up by the server
        pass
    
    def request(self, method: str, url: str, **kwargs) -> requests.Response:
        """Make a request using this session."""
        kwargs['session_id'] = self.session_id
        return self.client.request(method, url, **kwargs)
    
    def get(self, url: str, **kwargs) -> requests.Response:
        """Make a GET request using this session."""
        return self.request("GET", url, **kwargs)
    
    def post(self, url: str, **kwargs) -> requests.Response:
        """Make a POST request using this session."""
        return self.request("POST", url, **kwargs)
    
    def put(self, url: str, **kwargs) -> requests.Response:
        """Make a PUT request using this session."""
        return self.request("PUT", url, **kwargs)
    
    def patch(self, url: str, **kwargs) -> requests.Response:
        """Make a PATCH request using this session."""
        return self.request("PATCH", url, **kwargs)
    
    def delete(self, url: str, **kwargs) -> requests.Response:
        """Make a DELETE request using this session."""
        return self.request("DELETE", url, **kwargs)


class AsyncCycleTLSClient:
    """
    Async version of CycleTLS client using aiohttp.
    
    Examples:
        >>> async with AsyncCycleTLSClient() as client:
        ...     response = await client.get("https://httpbin.org/json")
        ...     data = await response.json()
    """
    
    def __init__(self, proxy_url: str = "http://localhost:8080", 
                 default_timeout: int = 30):
        if not AIOHTTP_AVAILABLE:
            raise ImportError("aiohttp is required for AsyncCycleTLSClient")
        
        self.proxy_url = proxy_url.rstrip('/')
        self.default_timeout = default_timeout
        self.session = None
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    def _make_headers(self, config: ProxyConfig, extra_headers: Optional[Dict[str, str]] = None) -> Dict[str, str]:
        """Create headers for the proxy request."""
        headers = {
            "X-URL": config.url,
            "X-IDENTIFIER": config.profile,
        }
        
        if config.session_id:
            headers["X-SESSION-ID"] = config.session_id
        
        if config.upstream_proxy:
            headers["X-PROXY"] = config.upstream_proxy
        
        if config.timeout != self.default_timeout:
            headers["X-TIMEOUT"] = str(config.timeout)
        
        if extra_headers:
            headers.update(extra_headers)
        
        return headers
    
    async def request(self, method: str, url: str,
                     profile: Union[str, BrowserProfile] = BrowserProfile.CHROME,
                     session_id: Optional[str] = None,
                     upstream_proxy: Optional[str] = None,
                     timeout: Optional[int] = None,
                     headers: Optional[Dict[str, str]] = None,
                     **kwargs) -> aiohttp.ClientResponse:
        """Make an async request through the CycleTLS-Proxy."""
        if not self.session:
            raise RuntimeError("Client must be used in async context manager")
        
        config = ProxyConfig(
            url=url,
            profile=profile,
            session_id=session_id,
            upstream_proxy=upstream_proxy,
            timeout=timeout or self.default_timeout
        )
        
        proxy_headers = self._make_headers(config, headers)
        
        try:
            timeout_obj = aiohttp.ClientTimeout(total=config.timeout + 5)
            async with self.session.request(
                method=method,
                url=self.proxy_url,
                headers=proxy_headers,
                timeout=timeout_obj,
                **kwargs
            ) as response:
                return response
        
        except asyncio.TimeoutError:
            raise CycleTLSTimeoutError(f"Request timed out after {config.timeout}s")
        except aiohttp.ClientError as e:
            raise CycleTLSError(f"Request error: {e}")
    
    async def get(self, url: str, **kwargs) -> aiohttp.ClientResponse:
        """Make an async GET request."""
        return await self.request("GET", url, **kwargs)
    
    async def post(self, url: str, **kwargs) -> aiohttp.ClientResponse:
        """Make an async POST request."""
        return await self.request("POST", url, **kwargs)


def run_basic_examples():
    """Run basic usage examples."""
    print("🚀 CycleTLS-Proxy Python Client Examples")
    print("=" * 50)
    
    client = CycleTLSClient()
    
    # Health check
    print("\n📊 Health Check")
    try:
        health = client.health_check()
        print(f"✓ Server status: {health['status']}")
        print(f"✓ Available profiles: {health['proxy']['profiles_available']}")
        print(f"✓ Active sessions: {health['proxy']['active_sessions']}")
    except Exception as e:
        print(f"✗ Health check failed: {e}")
        return
    
    print("\n🌐 Basic GET Requests with Different Profiles")
    
    # Test different browser profiles
    profiles_to_test = [
        (BrowserProfile.CHROME, "Chrome Linux"),
        (BrowserProfile.FIREFOX, "Firefox Linux"),
        (BrowserProfile.SAFARI_IOS, "Safari iOS"),
        (BrowserProfile.EDGE, "Microsoft Edge"),
    ]
    
    for profile, description in profiles_to_test:
        try:
            response = client.get("https://httpbin.org/user-agent", profile=profile)
            user_agent = response.json()["user-agent"]
            print(f"✓ {description}: {user_agent}")
        except Exception as e:
            print(f"✗ {description}: {e}")
    
    print("\n📝 HTTP Methods")
    
    # POST request with JSON
    try:
        data = {
            "username": "testuser",
            "email": "test@example.com",
            "active": True,
            "metadata": {"source": "python-example"}
        }
        response = client.post(
            "https://httpbin.org/post",
            profile=BrowserProfile.CHROME,
            json=data,
            headers={"Content-Type": "application/json"}
        )
        result = response.json()
        print(f"✓ POST with JSON: {result['json']['username']}")
    except Exception as e:
        print(f"✗ POST request failed: {e}")
    
    # PUT request
    try:
        response = client.put(
            "https://httpbin.org/put",
            profile=BrowserProfile.FIREFOX,
            data="Updated content",
            headers={"Content-Type": "text/plain"}
        )
        print(f"✓ PUT request: Status {response.status_code}")
    except Exception as e:
        print(f"✗ PUT request failed: {e}")
    
    # DELETE request
    try:
        response = client.delete(
            "https://httpbin.org/delete",
            profile=BrowserProfile.SAFARI
        )
        print(f"✓ DELETE request: Status {response.status_code}")
    except Exception as e:
        print(f"✗ DELETE request failed: {e}")


def run_session_examples():
    """Run session management examples."""
    print("\n🔄 Session Management")
    
    client = CycleTLSClient()
    
    # Session-based requests
    with client.session("demo-session") as session:
        try:
            # Set a cookie
            session.get("https://httpbin.org/cookies/set/session_token/abc123")
            
            # Verify cookie persistence
            response = session.get("https://httpbin.org/cookies")
            cookies = response.json().get("cookies", {})
            if "session_token" in cookies:
                print(f"✓ Session cookie persisted: {cookies['session_token']}")
            else:
                print("✗ Session cookie not found")
            
            # Add another cookie
            session.get("https://httpbin.org/cookies/set/user_id/12345")
            
            # Check both cookies
            response = session.get("https://httpbin.org/cookies")
            cookies = response.json().get("cookies", {})
            print(f"✓ Session has {len(cookies)} cookies: {list(cookies.keys())}")
            
        except Exception as e:
            print(f"✗ Session example failed: {e}")


def run_authentication_examples():
    """Run authentication examples."""
    print("\n🔐 Authentication Examples")
    
    client = CycleTLSClient()
    
    # Basic authentication
    try:
        import base64
        credentials = base64.b64encode(b"testuser:testpass").decode('ascii')
        response = client.get(
            "https://httpbin.org/basic-auth/testuser/testpass",
            headers={"Authorization": f"Basic {credentials}"},
            profile=BrowserProfile.CHROME
        )
        result = response.json()
        print(f"✓ Basic auth: {result['authenticated']}")
    except Exception as e:
        print(f"✗ Basic auth failed: {e}")
    
    # Bearer token authentication
    try:
        response = client.get(
            "https://httpbin.org/bearer",
            headers={"Authorization": "Bearer test-token-12345"},
            profile=BrowserProfile.FIREFOX
        )
        result = response.json()
        print(f"✓ Bearer token: {result['authenticated']}")
    except Exception as e:
        print(f"✗ Bearer token failed: {e}")


def run_advanced_examples():
    """Run advanced usage examples."""
    print("\n🚀 Advanced Features")
    
    client = CycleTLSClient()
    
    # Custom headers
    try:
        headers = {
            "X-API-Key": "secret-key-123",
            "X-Client-Version": "1.0.0",
            "Accept": "application/json",
            "User-Agent": "This-Should-Be-Overridden/1.0"
        }
        response = client.get(
            "https://httpbin.org/headers",
            headers=headers,
            profile=BrowserProfile.SAFARI_IOS
        )
        result_headers = response.json()["headers"]
        print(f"✓ Custom headers forwarded: {len(result_headers)} headers")
        
        # Check if User-Agent was correctly overridden by profile
        if "iPhone" in result_headers.get("User-Agent", ""):
            print("✓ Profile User-Agent correctly used")
        else:
            print("✗ Custom User-Agent incorrectly used")
            
    except Exception as e:
        print(f"✗ Custom headers failed: {e}")
    
    # Timeout configuration
    try:
        start_time = time.time()
        response = client.get(
            "https://httpbin.org/delay/1",
            timeout=5,
            profile=BrowserProfile.CHROME
        )
        duration = time.time() - start_time
        print(f"✓ Timeout test passed: {duration:.2f}s")
    except Exception as e:
        print(f"✗ Timeout test failed: {e}")
    
    # Large JSON payload
    try:
        large_data = {
            "items": [
                {"id": i, "name": f"Item {i}", "active": True}
                for i in range(100)
            ],
            "metadata": {
                "total": 100,
                "generated": time.time(),
                "client": "python-example"
            }
        }
        response = client.post(
            "https://httpbin.org/post",
            json=large_data,
            profile=BrowserProfile.EDGE
        )
        result = response.json()
        items_count = len(result["json"]["items"])
        print(f"✓ Large JSON payload: {items_count} items sent")
    except Exception as e:
        print(f"✗ Large JSON payload failed: {e}")


def run_error_handling_examples():
    """Run error handling examples."""
    print("\n⚠️  Error Handling")
    
    client = CycleTLSClient()
    
    # Invalid profile
    try:
        client.get("https://httpbin.org/get", profile="invalid-profile")
        print("✗ Invalid profile should have failed")
    except CycleTLSInvalidProfileError:
        print("✓ Invalid profile error handled correctly")
    except Exception as e:
        print(f"✗ Unexpected error for invalid profile: {e}")
    
    # Missing URL (should be caught by client)
    try:
        client.get("", profile=BrowserProfile.CHROME)
        print("✗ Empty URL should have failed")
    except Exception:
        print("✓ Empty URL error handled correctly")
    
    # Invalid URL
    try:
        client.get("not-a-url", profile=BrowserProfile.CHROME)
        print("✗ Invalid URL should have failed")
    except Exception:
        print("✓ Invalid URL error handled correctly")
    
    # Timeout error
    try:
        client.get("https://httpbin.org/delay/5", timeout=1, profile=BrowserProfile.CHROME)
        print("✗ Timeout should have occurred")
    except CycleTLSTimeoutError:
        print("✓ Timeout error handled correctly")
    except Exception as e:
        print(f"✗ Unexpected timeout error: {e}")


def run_concurrent_examples():
    """Run concurrent request examples."""
    print("\n🔄 Concurrent Requests")
    
    client = CycleTLSClient()
    
    def make_request(request_id):
        """Make a single request with unique session."""
        try:
            response = client.get(
                f"https://httpbin.org/get?request_id={request_id}",
                profile=BrowserProfile.CHROME,
                session_id=f"concurrent-{request_id}"
            )
            return request_id, response.status_code, len(response.content)
        except Exception as e:
            return request_id, "ERROR", str(e)
    
    # Run 10 concurrent requests
    start_time = time.time()
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [executor.submit(make_request, i) for i in range(10)]
        results = [future.result() for future in concurrent.futures.as_completed(futures)]
    
    duration = time.time() - start_time
    successful = sum(1 for _, status, _ in results if isinstance(status, int) and status == 200)
    
    print(f"✓ Completed {successful}/10 concurrent requests in {duration:.2f}s")
    for req_id, status, size in sorted(results):
        if isinstance(status, int):
            print(f"  Request {req_id}: HTTP {status}, {size} bytes")
        else:
            print(f"  Request {req_id}: {status}")


async def run_async_examples():
    """Run async client examples."""
    if not AIOHTTP_AVAILABLE:
        print("\n⚠️  Async examples require aiohttp (pip install aiohttp)")
        return
    
    print("\n⚡ Async Client Examples")
    
    async with AsyncCycleTLSClient() as client:
        # Basic async request
        try:
            async with await client.get(
                "https://httpbin.org/json",
                profile=BrowserProfile.CHROME
            ) as response:
                data = await response.json()
                print(f"✓ Async GET: {data['slideshow']['title']}")
        except Exception as e:
            print(f"✗ Async GET failed: {e}")
        
        # Multiple async requests
        try:
            tasks = []
            for i in range(5):
                task = client.get(
                    f"https://httpbin.org/get?async_req={i}",
                    profile=BrowserProfile.FIREFOX,
                    session_id=f"async-session-{i}"
                )
                tasks.append(task)
            
            start_time = time.time()
            responses = await asyncio.gather(*tasks)
            duration = time.time() - start_time
            
            print(f"✓ Completed 5 async requests in {duration:.2f}s")
            for i, response in enumerate(responses):
                async with response as resp:
                    data = await resp.json()
                    args = data.get('args', {})
                    print(f"  Request {i}: {args.get('async_req', 'N/A')}")
                    
        except Exception as e:
            print(f"✗ Multiple async requests failed: {e}")


def run_real_world_examples():
    """Run real-world usage examples."""
    print("\n🌍 Real-World Examples")
    
    client = CycleTLSClient()
    
    # Simulate login flow
    print("\n🔐 Simulated Login Flow")
    try:
        with client.session("login-demo") as session:
            # Get login page (simulate)
            login_page = session.get(
                "https://httpbin.org/get",
                profile=BrowserProfile.CHROME
            )
            print("✓ Step 1: Retrieved login page")
            
            # Submit login credentials
            login_data = {
                "username": "demo_user",
                "password": "secure_password",
                "remember_me": True
            }
            login_response = session.post(
                "https://httpbin.org/post",
                json=login_data,
                profile=BrowserProfile.CHROME,
                headers={"Content-Type": "application/json"}
            )
            print("✓ Step 2: Submitted login credentials")
            
            # Access protected resource
            protected_response = session.get(
                "https://httpbin.org/get?protected=true",
                profile=BrowserProfile.CHROME,
                headers={"Authorization": "Bearer fake-jwt-token"}
            )
            print("✓ Step 3: Accessed protected resource")
            
    except Exception as e:
        print(f"✗ Login flow failed: {e}")
    
    # API interaction example
    print("\n📡 API Interaction Example")
    try:
        # Create resource
        create_data = {
            "name": "Test Resource",
            "description": "Created via CycleTLS-Proxy",
            "active": True,
            "tags": ["test", "example", "python"]
        }
        create_response = client.post(
            "https://httpbin.org/post",
            json=create_data,
            profile=BrowserProfile.CHROME,
            headers={
                "Content-Type": "application/json",
                "X-API-Key": "demo-api-key-123"
            }
        )
        print("✓ Created resource via API")
        
        # Update resource
        update_data = {"name": "Updated Test Resource"}
        update_response = client.patch(
            "https://httpbin.org/patch",
            json=update_data,
            profile=BrowserProfile.CHROME,
            headers={
                "Content-Type": "application/json",
                "X-API-Key": "demo-api-key-123"
            }
        )
        print("✓ Updated resource via API")
        
        # List resources
        list_response = client.get(
            "https://httpbin.org/get?page=1&limit=10",
            profile=BrowserProfile.CHROME,
            headers={"X-API-Key": "demo-api-key-123"}
        )
        print("✓ Retrieved resource list via API")
        
    except Exception as e:
        print(f"✗ API interaction failed: {e}")


def main():
    """Run all examples."""
    print("CycleTLS-Proxy Python Client - Comprehensive Examples")
    print("=" * 60)
    
    try:
        # Basic examples
        run_basic_examples()
        
        # Session management
        run_session_examples()
        
        # Authentication
        run_authentication_examples()
        
        # Advanced features
        run_advanced_examples()
        
        # Error handling
        run_error_handling_examples()
        
        # Concurrent requests
        run_concurrent_examples()
        
        # Async examples
        asyncio.run(run_async_examples())
        
        # Real-world examples
        run_real_world_examples()
        
        print("\n🎉 All Examples Completed Successfully!")
        print("\nTo use this as a library:")
        print("```python")
        print("from examples.python import CycleTLSClient, BrowserProfile")
        print("")
        print("client = CycleTLSClient()")
        print("response = client.get('https://api.example.com', profile=BrowserProfile.CHROME)")
        print("print(response.json())")
        print("```")
        
    except KeyboardInterrupt:
        print("\n\n⏹️  Examples interrupted by user")
    except Exception as e:
        print(f"\n❌ Examples failed: {e}")


if __name__ == "__main__":
    main()