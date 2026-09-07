package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var testOptions = SecurityHeadersOptions{
	COEP: "require-corp",
	COOP: "same-origin",
	CORP: "same-origin",
}

func TestSecurityMiddlewareWithOptions(t *testing.T) {
	handler := SecurityMiddlewareWithOptions(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}), testOptions)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	assertExpectedSecurityHeaders(t, w, testOptions)
}

func TestSecurityMiddlewareFuncWithOptions(t *testing.T) {
	handler := SecurityMiddlewareFuncWithOptions(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, testOptions)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	assertExpectedSecurityHeaders(t, w, testOptions)
}

func TestMethodValidationMiddlewareWithOptions(t *testing.T) {
	handler := MethodValidationMiddlewareWithOptions(testOptions, "GET", "POST")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("allowed method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		assertExpectedSecurityHeaders(t, w, testOptions)
	})

	t.Run("disallowed method", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("Expected status 405, got %d", w.Code)
		}
		if allow := w.Header().Get("Allow"); allow != "GET, POST" {
			t.Fatalf("Expected Allow header 'GET, POST', got %q", allow)
		}

		assertExpectedSecurityHeaders(t, w, testOptions)
	})
}

func TestMethodValidationMiddlewareFuncWithOptions(t *testing.T) {
	handler := MethodValidationMiddlewareFuncWithOptions(testOptions, "GET")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("disallowed method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("Expected status 405, got %d", w.Code)
		}
		if allow := w.Header().Get("Allow"); allow != "GET" {
			t.Fatalf("Expected Allow header 'GET', got %q", allow)
		}
	})
}

func TestSecureHandlerWithOptions(t *testing.T) {
	handler := SecureHandlerWithOptions([]string{"GET", "HEAD"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, testOptions)

	t.Run("allowed method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		assertExpectedSecurityHeaders(t, w, testOptions)
	})

	t.Run("disallowed method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("Expected status 405, got %d", w.Code)
		}
		if allow := w.Header().Get("Allow"); allow != "GET, HEAD" {
			t.Fatalf("Expected Allow header 'GET, HEAD', got %q", allow)
		}

		assertExpectedSecurityHeaders(t, w, testOptions)
	})
}

func TestSetSecurityHeadersWithOptions(t *testing.T) {
	w := httptest.NewRecorder()
	setSecurityHeadersWithOptions(w, testOptions)
	assertExpectedSecurityHeaders(t, w, testOptions)
}

func assertExpectedSecurityHeaders(t *testing.T, w *httptest.ResponseRecorder, options SecurityHeadersOptions) {
	t.Helper()

	expectedHeaders := map[string]string{
		"Cache-Control":                     "no-cache, no-store, must-revalidate, private",
		"Content-Security-Policy":           "default-src 'none'; frame-ancestors 'none'",
		"Expires":                           "0",
		"Pragma":                            "no-cache",
		"Referrer-Policy":                   "strict-origin-when-cross-origin",
		"Server":                            "istio-test",
		"X-Content-Type-Options":            "nosniff",
		"X-Frame-Options":                   "DENY",
		"X-Permitted-Cross-Domain-Policies": "none",
		"X-Xss-Protection":                  "1; mode=block",
	}

	for header, expected := range expectedHeaders {
		if actual := w.Header().Get(header); actual != expected {
			t.Fatalf("Expected header %s to be %q, got %q", header, expected, actual)
		}
	}

	if options.COEP != "" && w.Header().Get("Cross-Origin-Embedder-Policy") != options.COEP {
		t.Fatalf("Expected COEP header %q, got %q", options.COEP, w.Header().Get("Cross-Origin-Embedder-Policy"))
	}
	if options.COOP != "" && w.Header().Get("Cross-Origin-Opener-Policy") != options.COOP {
		t.Fatalf("Expected COOP header %q, got %q", options.COOP, w.Header().Get("Cross-Origin-Opener-Policy"))
	}
	if options.CORP != "" && w.Header().Get("Cross-Origin-Resource-Policy") != options.CORP {
		t.Fatalf("Expected CORP header %q, got %q", options.CORP, w.Header().Get("Cross-Origin-Resource-Policy"))
	}
}
