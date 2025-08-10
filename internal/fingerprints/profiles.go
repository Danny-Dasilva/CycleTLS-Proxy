// Package fingerprints provides TLS fingerprint profiles for browser emulation.
// This package contains comprehensive fingerprint data including JA3, JA4,
// User-Agent strings and other TLS characteristics for popular browsers.
package fingerprints

import (
	"fmt"
	"sort"
)

// Profile represents a comprehensive TLS fingerprint profile containing
// all necessary data to mimic a specific browser's TLS characteristics.
type Profile struct {
	// JA3 is the JA3 TLS fingerprint string
	JA3 string `json:"ja3"`
	
	// JA4 is the JA4 TLS fingerprint string (newer standard)
	JA4 string `json:"ja4"`
	
	// UserAgent is the HTTP User-Agent header string
	UserAgent string `json:"user_agent"`
	
	// HTTPVersion specifies the preferred HTTP version (e.g., "h2", "http/1.1")
	HTTPVersion string `json:"http_version"`
	
	// TLSVersion specifies the TLS version (e.g., "1.3", "1.2")
	TLSVersion string `json:"tls_version"`
	
	// Description provides human-readable information about the profile
	Description string `json:"description"`
	
	// Platform indicates the operating system/platform (e.g., "Windows", "macOS", "Linux", "iOS", "Android")
	Platform string `json:"platform"`
}

// GetDefaultProfiles returns a comprehensive map of TLS fingerprint profiles
// for major browsers with current and accurate fingerprint data.
func GetDefaultProfiles() map[string]Profile {
	return map[string]Profile{
		"chrome": {
			JA3:         "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,45-27-23-10-13-35-5-65037-16-51-0-18-43-11-17513-65281,29-23-24,0",
			JA4:         "t13d1517h2_8daaf6152771_7e51fdad25f2",
			UserAgent:   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Google Chrome 120 on Linux",
			Platform:    "Linux",
		},
		"chrome_windows": {
			JA3:         "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,45-27-23-10-13-35-5-65037-16-51-0-18-43-11-17513-65281-21,29-23-24,0",
			JA4:         "t13d1517h2_8daaf6152771_3c1b64d5e4f2",
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Google Chrome 120 on Windows 10/11",
			Platform:    "Windows",
		},
		"firefox": {
			JA3:         "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-156-157-47-53-10,0-23-65281-10-11-16-5-34-51-43-13-45-28-65037,29-23-24-25-256-257,0",
			JA4:         "t13d1717h2_5b57614c22b0_f2748d6cd58d",
			UserAgent:   "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Mozilla Firefox 121 on Ubuntu Linux",
			Platform:    "Linux",
		},
		"firefox_windows": {
			JA3:         "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-156-157-47-53-10,0-23-65281-10-11-16-5-34-51-43-13-45-28-65037-21,29-23-24-25-256-257,0",
			JA4:         "t13d1717h2_5b57614c22b0_8f2c4d6e5a3b",
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Mozilla Firefox 121 on Windows 10/11",
			Platform:    "Windows",
		},
		"safari_ios": {
			JA3:         "771,4865-4866-4867-49196-49195-52393-49200-49199-52392-49162-157-156-53-47-49160-49170-10,0-23-65281-10-11-16-5-13-18-51-45-43-27-21,29-23-24-25,0",
			JA4:         "t13d1516h2_8daaf6152771_b0da82dd1658",
			UserAgent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.1 Mobile/15E148 Safari/604.1",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Safari on iOS 17.1.1",
			Platform:    "iOS",
		},
		"safari": {
			JA3:         "771,4865-4866-4867-49196-49195-52393-49200-49199-52392-49162-49161-49171-49172-156-157-47-53,65281-0-23-13-5-18-16-30032-11-10-35-22-23,29-23-24,0",
			JA4:         "t13d1516h2_8daaf6152771_9b3e7c5a2f1d",
			UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Safari 17 on macOS",
			Platform:    "macOS",
		},
		"edge": {
			JA3:         "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,45-27-23-10-13-35-5-65037-16-51-0-18-43-11-17513-65281-28,29-23-24,0",
			JA4:         "t13d1517h2_8daaf6152771_c4f8a2d7e3b1",
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "Microsoft Edge 120 on Windows 10/11",
			Platform:    "Windows",
		},
		"okhttp": {
			JA3:         "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27,29-23-24,0",
			JA4:         "t13d1517h2_8daaf6152771_6a9c3e5f1b8d",
			UserAgent:   "okhttp/4.12.0",
			HTTPVersion: "h2",
			TLSVersion:  "1.3",
			Description: "OkHttp 4.12.0 Android HTTP client",
			Platform:    "Android",
		},
		"chrome_legacy_tls12": {
			JA3:         "771,49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-11-51-45-43-10-27-21,29-23-24,0",
			JA4:         "t12d1209h2_d34a8e72043a_b39be8c56a14",
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			HTTPVersion: "h2",
			TLSVersion:  "1.2",
			Description: "Chrome 91 with TLS 1.2 support",
			Platform:    "Windows",
		},
	}
}

// GetProfile returns a specific profile by identifier.
// Returns the profile and a boolean indicating if the profile was found.
func GetProfile(identifier string) (Profile, bool) {
	profiles := GetDefaultProfiles()
	profile, exists := profiles[identifier]
	return profile, exists
}

// ListProfiles returns a sorted list of all available profile identifiers.
func ListProfiles() []string {
	profiles := GetDefaultProfiles()
	identifiers := make([]string, 0, len(profiles))
	for identifier := range profiles {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)
	return identifiers
}

// ValidateProfile checks if a profile identifier is valid.
func ValidateProfile(identifier string) bool {
	_, exists := GetProfile(identifier)
	return exists
}

// GetProfilesByPlatform returns all profiles for a specific platform.
func GetProfilesByPlatform(platform string) map[string]Profile {
	allProfiles := GetDefaultProfiles()
	platformProfiles := make(map[string]Profile)
	
	for id, profile := range allProfiles {
		if profile.Platform == platform {
			platformProfiles[id] = profile
		}
	}
	return platformProfiles
}

// GetPlatforms returns a sorted list of all available platforms.
func GetPlatforms() []string {
	profiles := GetDefaultProfiles()
	platformSet := make(map[string]bool)
	
	for _, profile := range profiles {
		platformSet[profile.Platform] = true
	}
	
	platforms := make([]string, 0, len(platformSet))
	for platform := range platformSet {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)
	return platforms
}

// ProfileInfo returns formatted information about a profile.
func (p *Profile) String() string {
	return fmt.Sprintf("%s (%s) - TLS %s, %s", 
		p.Description, p.Platform, p.TLSVersion, p.HTTPVersion)
}

// Convenience functions for commonly used profiles

// Chrome returns the default Chrome profile (Linux).
func Chrome() Profile {
	profile, _ := GetProfile("chrome")
	return profile
}

// ChromeWindows returns the Chrome Windows profile.
func ChromeWindows() Profile {
	profile, _ := GetProfile("chrome_windows")
	return profile
}

// Firefox returns the default Firefox profile (Linux).
func Firefox() Profile {
	profile, _ := GetProfile("firefox")
	return profile
}

// FirefoxWindows returns the Firefox Windows profile.
func FirefoxWindows() Profile {
	profile, _ := GetProfile("firefox_windows")
	return profile
}

// SafariIOS returns the Safari iOS profile.
func SafariIOS() Profile {
	profile, _ := GetProfile("safari_ios")
	return profile
}

// Safari returns the Safari macOS profile.
func Safari() Profile {
	profile, _ := GetProfile("safari")
	return profile
}

// Edge returns the Microsoft Edge profile.
func Edge() Profile {
	profile, _ := GetProfile("edge")
	return profile
}

// OkHttp returns the OkHttp Android profile.
func OkHttp() Profile {
	profile, _ := GetProfile("okhttp")
	return profile
}

// ChromeLegacyTLS12 returns a Chrome profile with TLS 1.2 support.
func ChromeLegacyTLS12() Profile {
	profile, _ := GetProfile("chrome_legacy_tls12")
	return profile
}