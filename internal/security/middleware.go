// Package security provides HTTP security middleware with configurable Cross-Origin policies.
package security

import (
	"net/http"
	"strings"
)

// SecurityHeadersOptions defines configurable security header policies.
type SecurityHeadersOptions struct {
	COEP string // Cross-Origin-Embedder-Policy: "", "require-corp", or "credentialless"
	COOP string // Cross-Origin-Opener-Policy: "same-origin", "same-origin-allow-popups", or "unsafe-none"
	CORP string // Cross-Origin-Resource-Policy: "same-origin", "same-site", or "cross-origin"
}

// SecurityMiddlewareWithOptions wraps an HTTP handler with configurable security headers.
func SecurityMiddlewareWithOptions(next http.Handler, options SecurityHeadersOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeadersWithOptions(w, options)
		next.ServeHTTP(w, r)
	})
}

// SecurityMiddlewareFuncWithOptions wraps an HTTP handler function with configurable security headers.
func SecurityMiddlewareFuncWithOptions(next http.HandlerFunc, options SecurityHeadersOptions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeadersWithOptions(w, options)
		next.ServeHTTP(w, r)
	}
}

// MethodValidationMiddlewareWithOptions ensures only specified HTTP methods are allowed with configurable security headers.
func MethodValidationMiddlewareWithOptions(options SecurityHeadersOptions, allowedMethods ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			methodAllowed := false
			for _, method := range allowedMethods {
				if r.Method == method {
					methodAllowed = true
					break
				}
			}

			if !methodAllowed {
				setSecurityHeadersWithOptions(w, options)
				w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			setSecurityHeadersWithOptions(w, options)
			next.ServeHTTP(w, r)
		})
	}
}

// MethodValidationMiddlewareFuncWithOptions ensures only specified HTTP methods are allowed for handler functions with configurable security headers.
func MethodValidationMiddlewareFuncWithOptions(options SecurityHeadersOptions, allowedMethods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			methodAllowed := false
			for _, method := range allowedMethods {
				if r.Method == method {
					methodAllowed = true
					break
				}
			}

			if !methodAllowed {
				setSecurityHeadersWithOptions(w, options)
				w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			setSecurityHeadersWithOptions(w, options)
			next.ServeHTTP(w, r)
		}
	}
}

// setSecurityHeadersWithOptions adds security headers based on the provided options.
func setSecurityHeadersWithOptions(w http.ResponseWriter, options SecurityHeadersOptions) {
	headers := w.Header()

	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
	headers.Set("X-XSS-Protection", "1; mode=block")
	headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	headers.Set("X-Permitted-Cross-Domain-Policies", "none")
	headers.Set("Server", "istio-test")
	headers.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	headers.Set("Cache-Control", "no-cache, no-store, must-revalidate, private")
	headers.Set("Pragma", "no-cache")
	headers.Set("Expires", "0")

	if options.COEP != "" {
		headers.Set("Cross-Origin-Embedder-Policy", options.COEP)
	}
	if options.COOP != "" {
		headers.Set("Cross-Origin-Opener-Policy", options.COOP)
	}
	if options.CORP != "" {
		headers.Set("Cross-Origin-Resource-Policy", options.CORP)
	}
}

// SecureHandlerWithOptions wraps a handler function with both security headers and method validation using configurable security options.
func SecureHandlerWithOptions(allowedMethods []string, handler http.HandlerFunc, options SecurityHeadersOptions) http.HandlerFunc {
	return MethodValidationMiddlewareFuncWithOptions(options, allowedMethods...)(SecurityMiddlewareFuncWithOptions(handler, options))
}
