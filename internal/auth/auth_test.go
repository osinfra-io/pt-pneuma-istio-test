package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"istio-test/internal/security"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		statusCode int
		identity   Identity
	}{
		{
			name: "authenticated identity",
			headers: map[string]string{
				"X-Authentik-Username": "alice",
				"X-Authentik-Email":    "alice@example.com",
				"X-Authentik-Name":     "Alice Example",
				"X-Authentik-Uid":      "user-123",
				"X-Authentik-Groups":   "all|platform| all ",
			},
			statusCode: http.StatusOK,
			identity: Identity{
				Username: "alice",
				Email:    "alice@example.com",
				Name:     "Alice Example",
				UID:      "user-123",
				Groups:   []string{"all", "platform", "all"},
			},
		},
		{
			name:       "missing identity",
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/istio-test/auth", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()

			Handler(w, req)

			if w.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, w.Code)
			}
			if tt.statusCode != http.StatusOK {
				return
			}

			var actual Identity
			if err := json.NewDecoder(w.Body).Decode(&actual); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if actual.Username != tt.identity.Username || actual.Email != tt.identity.Email || actual.Name != tt.identity.Name || actual.UID != tt.identity.UID {
				t.Fatalf("unexpected identity: %#v", actual)
			}
			if len(actual.Groups) != len(tt.identity.Groups) {
				t.Fatalf("expected groups %#v, got %#v", tt.identity.Groups, actual.Groups)
			}
			for i := range actual.Groups {
				if actual.Groups[i] != tt.identity.Groups[i] {
					t.Fatalf("expected groups %#v, got %#v", tt.identity.Groups, actual.Groups)
				}
			}
		})
	}
}

func TestHandlerIncludesRequestContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/istio-test/auth", nil)
	req.Host = "pneuma.sb.osinfra.io"
	req.Header.Set("X-Authentik-Username", "alice")
	req.Header.Set("X-Authentik-Jwt", "secret-token")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Authentik-Meta-App", "pneuma-sb-osinfra-io")
	req.Header.Set("X-Authentik-Meta-Provider", "provider")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Request-Id", "req-1")
	req.Header.Set("Traceparent", "00-abc-def-01")
	req.Header.Set("X-Forwarded-Client-Cert", "By=spiffe://cluster.local/ns/a/sa/b;Hash=abc;URI=spiffe://cluster.local/ns/istio-system/sa/gateway")
	w := httptest.NewRecorder()

	Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret-token") {
		t.Fatalf("response must not expose token values: %s", w.Body.String())
	}

	var actual Response
	if err := json.NewDecoder(w.Body).Decode(&actual); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if actual.Request.Host != "pneuma.sb.osinfra.io" || actual.Request.ForwardedProto != "https" ||
		actual.Request.RequestID != "req-1" || actual.Request.Traceparent != "00-abc-def-01" ||
		actual.Request.PeerIdentity != "spiffe://cluster.local/ns/istio-system/sa/gateway" {
		t.Fatalf("unexpected request context: %#v", actual.Request)
	}
	if actual.Authentik.Application != "pneuma-sb-osinfra-io" || actual.Authentik.Provider != "provider" {
		t.Fatalf("unexpected authentik context: %#v", actual.Authentik)
	}
	if len(actual.Request.AuthentikHeaders) != 4 || actual.Request.AuthentikHeaders[0] != "X-Authentik-Jwt" {
		t.Fatalf("expected sorted authentik header names, got %#v", actual.Request.AuthentikHeaders)
	}
	if actual.Workload.Version == "" {
		t.Fatalf("expected workload version")
	}
}

func TestSecureHandlerWithOptions(t *testing.T) {
	handler := SecureHandlerWithOptions(security.SecurityHeadersOptions{COEP: "require-corp", COOP: "same-origin", CORP: "same-origin"})

	t.Run("allows get", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/istio-test/auth", nil)
		req.Header.Set("X-Authentik-Username", "alice")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected JSON content type, got %q", got)
		}
		if got := w.Header().Get("Server"); got != "istio-test" {
			t.Fatalf("expected security headers, got Server %q", got)
		}
	})

	t.Run("rejects post", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/istio-test/auth", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected status 405, got %d", w.Code)
		}
		if got := w.Header().Get("Allow"); got != "GET, HEAD" {
			t.Fatalf("expected Allow header, got %q", got)
		}
	})
}

func TestSplitGroups(t *testing.T) {
	if got := splitGroups(" "); len(got) != 0 {
		t.Fatalf("expected no groups, got %#v", got)
	}
	if got := splitGroups("all|platform"); len(got) != 2 || got[0] != "all" || got[1] != "platform" {
		t.Fatalf("unexpected groups: %#v", got)
	}
}
