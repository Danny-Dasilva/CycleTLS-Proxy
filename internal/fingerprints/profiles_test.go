package fingerprints

import (
	"testing"
)

func TestGetDefaultProfiles(t *testing.T) {
	profiles := GetDefaultProfiles()

	if len(profiles) == 0 {
		t.Fatal("GetDefaultProfiles returned empty map")
	}

	// Check that essential profiles exist
	essentialProfiles := []string{"chrome", "firefox", "safari"}
	for _, profileName := range essentialProfiles {
		profile, exists := profiles[profileName]
		if !exists {
			t.Errorf("Essential profile '%s' not found", profileName)
			continue
		}

		// Validate profile has required fields
		if profile.UserAgent == "" {
			t.Errorf("Profile '%s' missing UserAgent", profileName)
		}

		// Profile should have either JA3 or JA4 (or both)
		if profile.JA3 == "" && profile.JA4 == "" {
			t.Errorf("Profile '%s' missing both JA3 and JA4 fingerprints", profileName)
		}
	}
}

func TestProfileStructure(t *testing.T) {
	profiles := GetDefaultProfiles()

	for name, profile := range profiles {
		t.Run(name, func(t *testing.T) {
			// Test UserAgent is not empty
			if profile.UserAgent == "" {
				t.Errorf("Profile %s has empty UserAgent", name)
			}

			// Test UserAgent contains reasonable browser info
			ua := profile.UserAgent
			if len(ua) < 10 {
				t.Errorf("Profile %s UserAgent seems too short: %s", name, ua)
			}

			// Test fingerprints format (basic validation)
			if profile.JA3 != "" {
				// JA3 should be a hash-like string
				if len(profile.JA3) < 32 {
					t.Errorf("Profile %s JA3 seems too short: %s", name, profile.JA3)
				}
			}

			if profile.JA4 != "" {
				// JA4 should follow format like t13d1516h2_8daaf6152771_b0da82dd1658
				if len(profile.JA4) < 20 {
					t.Errorf("Profile %s JA4 seems too short: %s", name, profile.JA4)
				}
			}
		})
	}
}

func BenchmarkGetDefaultProfiles(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiles := GetDefaultProfiles()
		if len(profiles) == 0 {
			b.Fatal("GetDefaultProfiles returned empty map")
		}
	}
}

func TestProfileUniqueness(t *testing.T) {
	profiles := GetDefaultProfiles()

	// Check that profiles have unique fingerprints
	ja3Map := make(map[string][]string)
	ja4Map := make(map[string][]string)

	for name, profile := range profiles {
		if profile.JA3 != "" {
			ja3Map[profile.JA3] = append(ja3Map[profile.JA3], name)
		}
		if profile.JA4 != "" {
			ja4Map[profile.JA4] = append(ja4Map[profile.JA4], name)
		}
	}

	// Check for JA3 duplicates
	for ja3, profileNames := range ja3Map {
		if len(profileNames) > 1 {
			t.Errorf("JA3 fingerprint %s is used by multiple profiles: %v", ja3, profileNames)
		}
	}

	// Check for JA4 duplicates
	for ja4, profileNames := range ja4Map {
		if len(profileNames) > 1 {
			t.Errorf("JA4 fingerprint %s is used by multiple profiles: %v", ja4, profileNames)
		}
	}
}