// Package auth provides handlers for displaying authenticated Authentik identity details.
package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"

	"istio-test/internal/metadata"
	"istio-test/internal/security"
)

const (
	usernameHeader = "X-Authentik-Username"
	emailHeader    = "X-Authentik-Email"
	nameHeader     = "X-Authentik-Name"
	uidHeader      = "X-Authentik-Uid"
	groupsHeader   = "X-Authentik-Groups"
)

const (
	authentikHeaderPrefix = "X-Authentik-"
	namespaceFile         = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// Request describes how the request reached the workload.
type Request struct {
	AuthentikHeaders []string `json:"authentikHeaders"`
	ForwardedFor     string   `json:"forwardedFor,omitempty"`
	ForwardedHost    string   `json:"forwardedHost,omitempty"`
	ForwardedProto   string   `json:"forwardedProto,omitempty"`
	Host             string   `json:"host"`
	PeerIdentity     string   `json:"peerIdentity,omitempty"`
	RequestID        string   `json:"requestId,omitempty"`
	Traceparent      string   `json:"traceparent,omitempty"`
}

// Authentik describes the application, provider and outpost that authorized the request.
type Authentik struct {
	Application  string `json:"application,omitempty"`
	Entitlements string `json:"entitlements,omitempty"`
	GoogleEmail  string `json:"googleEmail,omitempty"`
	Outpost      string `json:"outpost,omitempty"`
	Provider     string `json:"provider,omitempty"`
	Version      string `json:"version,omitempty"`
}

// Workload describes the pod that served the request.
type Workload struct {
	Namespace string `json:"namespace,omitempty"`
	Pod       string `json:"pod,omitempty"`
	Version   string `json:"version"`
}

// Identity contains the authenticated user's non-sensitive identity details.
type Identity struct {
	Username string   `json:"username,omitempty"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	UID      string   `json:"uid,omitempty"`
	Groups   []string `json:"groups"`
}

// Response is the identity plus the troubleshooting context for the request.
type Response struct {
	Identity
	Authentik Authentik `json:"authentik"`
	Request   Request   `json:"request"`
	Workload  Workload  `json:"workload"`
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

	response := Response{
		Identity: identity,
		Authentik: Authentik{
			Application:  r.Header.Get("X-Authentik-Meta-App"),
			Entitlements: r.Header.Get("X-Authentik-Entitlements"),
			GoogleEmail:  r.Header.Get("X-Authentik-Osinfra-Google-Email"),
			Outpost:      r.Header.Get("X-Authentik-Meta-Outpost"),
			Provider:     r.Header.Get("X-Authentik-Meta-Provider"),
			Version:      r.Header.Get("X-Authentik-Meta-Version"),
		},
		Request: Request{
			AuthentikHeaders: authentikHeaderNames(r.Header),
			ForwardedFor:     r.Header.Get("X-Forwarded-For"),
			ForwardedHost:    r.Header.Get("X-Forwarded-Host"),
			ForwardedProto:   r.Header.Get("X-Forwarded-Proto"),
			Host:             r.Host,
			PeerIdentity:     peerIdentity(r.Header.Get("X-Forwarded-Client-Cert")),
			RequestID:        r.Header.Get("X-Request-Id"),
			Traceparent:      r.Header.Get("Traceparent"),
		},
		Workload: Workload{
			Namespace: namespace(),
			Pod:       os.Getenv("HOSTNAME"),
			Version:   metadata.Version(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode identity response", http.StatusInternalServerError)
	}
}

// authentikHeaderNames lists only the names of the identity headers received, never their values.
func authentikHeaderNames(headers http.Header) []string {
	names := make([]string, 0)
	for name := range headers {
		if strings.HasPrefix(name, authentikHeaderPrefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// peerIdentity extracts only the caller's SPIFFE ID from the last hop of the forwarded client
// certificate header. Entries are separated by commas and fields by semicolons, but quoted values
// (such as Subject) may contain either, so splitting honors double quotes.
func peerIdentity(xfcc string) string {
	entries := splitOutsideQuotes(xfcc, ',')
	if len(entries) == 0 {
		return ""
	}
	for _, field := range splitOutsideQuotes(entries[len(entries)-1], ';') {
		key, value, found := strings.Cut(field, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "URI") {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if strings.HasPrefix(value, "spiffe://") {
			return value
		}
	}
	return ""
}

func splitOutsideQuotes(s string, separator rune) []string {
	var parts []string
	var current strings.Builder
	quoted := false
	for _, r := range s {
		switch {
		case r == '"':
			quoted = !quoted
			current.WriteRune(r)
		case r == separator && !quoted:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	return append(parts, current.String())
}

func namespace() string {
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
