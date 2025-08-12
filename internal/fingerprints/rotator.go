// Package fingerprints provides profile rotation functionality for automatic
// browser fingerprint switching with multiple rotation strategies.
package fingerprints

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
)

// RotationStrategy defines the rotation strategy type
type RotationStrategy string

const (
	RotationRandom RotationStrategy = "random"
)

// RotationConfig contains configuration for profile rotation
type RotationConfig struct {
	// EnabledProfiles lists the profiles that should be rotated through
	EnabledProfiles []string `json:"enabled_profiles"`
	
	// RotationEnabled determines if random rotation is active
	RotationEnabled bool `json:"rotation_enabled"`
	
	// SessionSticky ensures the same session uses the same profile
	SessionSticky bool `json:"session_sticky"`
}

// DefaultRotationConfig returns a sensible default configuration
func DefaultRotationConfig() *RotationConfig {
	return &RotationConfig{
		EnabledProfiles: []string{"chrome138", "chrome139"},
		RotationEnabled: true,
		SessionSticky:   true,
	}
}

// ProfileRotator manages automatic rotation of browser profiles
type ProfileRotator struct {
	config          *RotationConfig
	mu              sync.RWMutex
	sessionProfiles map[string]string // sessionID -> profile
	allProfiles     map[string]Profile
}

// NewProfileRotator creates a new profile rotator with the given configuration
func NewProfileRotator(config *RotationConfig) *ProfileRotator {
	if config == nil {
		config = DefaultRotationConfig()
	}
	
	return &ProfileRotator{
		config:          config,
		sessionProfiles: make(map[string]string),
		allProfiles:     GetDefaultProfiles(),
	}
}

// UpdateConfig updates the rotation configuration
func (r *ProfileRotator) UpdateConfig(config *RotationConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.config = config
	// Clear session profiles if session sticky changed
	if !config.SessionSticky {
		r.sessionProfiles = make(map[string]string)
	}
}

// GetConfig returns the current rotation configuration
func (r *ProfileRotator) GetConfig() *RotationConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a copy to prevent external modification
	configCopy := *r.config
	configCopy.EnabledProfiles = make([]string, len(r.config.EnabledProfiles))
	copy(configCopy.EnabledProfiles, r.config.EnabledProfiles)
	
	return &configCopy
}

// GetProfileForSession returns the appropriate profile for a session
// If sessionID is empty, returns a profile based on random rotation (if enabled)
func (r *ProfileRotator) GetProfileForSession(sessionID string) (Profile, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var profileID string
	
	// If rotation is disabled, use first enabled profile or fallback
	if !r.config.RotationEnabled {
		if len(r.config.EnabledProfiles) > 0 {
			profileID = r.config.EnabledProfiles[0]
		} else {
			profileID = "chrome138" // Default fallback
		}
	} else {
		// Check if session should use sticky profile
		if r.config.SessionSticky && sessionID != "" {
			if existingProfile, exists := r.sessionProfiles[sessionID]; exists {
				if profile, profileExists := r.allProfiles[existingProfile]; profileExists {
					return profile, existingProfile, nil
				}
				// Profile no longer exists, remove from session mapping
				delete(r.sessionProfiles, sessionID)
			}
		}
		
		// Select random profile from enabled profiles
		profileID = r.selectRandomProfile()
	}
	
	// Store session profile mapping
	if r.config.SessionSticky && sessionID != "" {
		r.sessionProfiles[sessionID] = profileID
	}
	
	// Get the profile
	if profile, exists := r.allProfiles[profileID]; exists {
		return profile, profileID, nil
	}
	
	// Fallback to first enabled profile if selected profile doesn't exist
	if len(r.config.EnabledProfiles) > 0 {
		fallbackID := r.config.EnabledProfiles[0]
		if profile, exists := r.allProfiles[fallbackID]; exists {
			return profile, fallbackID, nil
		}
	}
	
	// Final fallback to chrome138
	if profile, exists := r.allProfiles["chrome138"]; exists {
		return profile, "chrome138", nil
	}
	
	// If all else fails, return chrome profile
	profile, exists := r.allProfiles["chrome"]
	if !exists {
		// This should never happen with valid profiles
		return Profile{}, "", fmt.Errorf("no valid profiles available")
	}
	
	return profile, "chrome", nil
}


// selectRandomProfile selects a random profile from enabled profiles
func (r *ProfileRotator) selectRandomProfile() string {
	if len(r.config.EnabledProfiles) == 0 {
		return "chrome138"
	}
	
	if len(r.config.EnabledProfiles) == 1 {
		return r.config.EnabledProfiles[0]
	}
	
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(r.config.EnabledProfiles))))
	if err != nil {
		// Fallback to first profile if random generation fails
		return r.config.EnabledProfiles[0]
	}
	
	return r.config.EnabledProfiles[n.Int64()]
}

// GetEnabledProfiles returns the list of currently enabled profiles
func (r *ProfileRotator) GetEnabledProfiles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	enabled := make([]string, len(r.config.EnabledProfiles))
	copy(enabled, r.config.EnabledProfiles)
	return enabled
}

// SetEnabledProfiles updates the list of enabled profiles
func (r *ProfileRotator) SetEnabledProfiles(profiles []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.config.EnabledProfiles = make([]string, len(profiles))
	copy(r.config.EnabledProfiles, profiles)
}

// AddProfile adds a profile to the enabled list
func (r *ProfileRotator) AddProfile(profileID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if profile already exists
	for _, existing := range r.config.EnabledProfiles {
		if existing == profileID {
			return // Already enabled
		}
	}
	
	r.config.EnabledProfiles = append(r.config.EnabledProfiles, profileID)
}

// RemoveProfile removes a profile from the enabled list
func (r *ProfileRotator) RemoveProfile(profileID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	newProfiles := make([]string, 0, len(r.config.EnabledProfiles))
	for _, existing := range r.config.EnabledProfiles {
		if existing != profileID {
			newProfiles = append(newProfiles, existing)
		}
	}
	
	r.config.EnabledProfiles = newProfiles
	
	// Remove from session mappings as well
	for sessionID, sessionProfile := range r.sessionProfiles {
		if sessionProfile == profileID {
			delete(r.sessionProfiles, sessionID)
		}
	}
}

// GetRotationStats returns statistics about the rotation state
func (r *ProfileRotator) GetRotationStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return map[string]interface{}{
		"rotation_enabled": r.config.RotationEnabled,
		"enabled_profiles": r.config.EnabledProfiles,
		"active_sessions":  len(r.sessionProfiles),
		"session_sticky":   r.config.SessionSticky,
	}
}

// ClearSessionMappings clears all session-to-profile mappings
func (r *ProfileRotator) ClearSessionMappings() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.sessionProfiles = make(map[string]string)
}

// SetRotationEnabled enables or disables random rotation
func (r *ProfileRotator) SetRotationEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.config.RotationEnabled = enabled
}

// IsRotationEnabled checks if rotation is enabled
func (r *ProfileRotator) IsRotationEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.config.RotationEnabled && len(r.config.EnabledProfiles) > 0
}

// GetProfileByID returns a profile by its identifier
func GetProfileByID(identifier string, rotator *ProfileRotator) (Profile, string, error) {
	// Handle rotation request
	if identifier == "auto-rotate" || identifier == "random" {
		if rotator != nil && rotator.IsRotationEnabled() {
			return rotator.GetProfileForSession("")
		}
		// Fallback to chrome138 if no rotator or rotation disabled
		if profile, exists := GetProfile("chrome138"); exists {
			return profile, "chrome138", nil
		}
		if profile, exists := GetProfile("chrome"); exists {
			return profile, "chrome", nil
		}
		return Profile{}, "", fmt.Errorf("no valid profiles available")
	}
	
	// Regular profile lookup
	if profile, exists := GetProfile(identifier); exists {
		return profile, identifier, nil
	}
	
	// Fallback to chrome if profile not found
	if profile, exists := GetProfile("chrome"); exists {
		return profile, "chrome", nil
	}
	
	return Profile{}, "", fmt.Errorf("profile '%s' not found", identifier)
}