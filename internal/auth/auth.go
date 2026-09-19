// Package auth provides handlers for displaying authenticated Authentik identity details.
package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"istio-test/internal/security"
)

const (
	usernameHeader = "X-Authentik-Username"
	emailHeader    = "X-Authentik-Email"
	nameHeader     = "X-Authentik-Name"
	uidHeader      = "X-Authentik-Uid"
	groupsHeader   = "X-Authentik-Groups"
)

// Identity contains the authenticated user's non-sensitive identity details.
type Identity struct {
	Username string   `json:"username,omitempty"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	UID      string   `json:"uid,omitempty"`
	Groups   []string `json:"groups"`
}

// Handler returns the identity details injected by Authentik's trusted forward-auth outpost.
func Handler(w http.ResponseWriter, r *http.Request) {
	identity := Identity{
		Username: r.Header.Get(usernameHeader),
		Email:    r.Header.Get(emailHeader),
		Name:     r.Header.Get(nameHeader),
		UID:      r.Header.Get(uidHeader),
		Groups:   splitGroups(r.Header.Get(groupsHeader)),
	}

	if identity.Username == "" && identity.Email == "" && identity.Name == "" && identity.UID == "" && len(identity.Groups) == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(identity); err != nil {
		http.Error(w, "Failed to encode identity response", http.StatusInternalServerError)
	}
}

func splitGroups(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}

	groups := make([]string, 0)
	for _, group := range strings.Split(value, "|") {
		if group = strings.TrimSpace(group); group != "" {
			groups = append(groups, group)
		}
	}
	return groups
}

// SecureHandlerWithOptions returns an identity handler with security headers and method validation.
func SecureHandlerWithOptions(options security.SecurityHeadersOptions) http.HandlerFunc {
	return security.SecureHandlerWithOptions([]string{"GET", "HEAD"}, Handler, options)
}
