package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/istio-test/metadata/", MetadataHandler(func(ctx context.Context, url string) (string, error) {
		switch url {
		case ClusterNameURL:
			return "test-cluster-name", nil
		case ClusterLocationURL:
			return "test-cluster-location", nil
		case InstanceZoneURL:
			return "projects/1234567890/zones/us-central1-a", nil
		default:
			return "", fmt.Errorf("unknown URL: %s", url)
		}
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
		jsonBody     bool
	}{
		{
			name:         "cluster name",
			path:         "/istio-test/metadata/cluster-name",
			expectedCode: http.StatusOK,
			expectedBody: `{"cluster-name":"test-cluster-name"}`,
			jsonBody:     true,
		},
		{
			name:         "cluster location",
			path:         "/istio-test/metadata/cluster-location",
			expectedCode: http.StatusOK,
			expectedBody: `{"cluster-location":"test-cluster-location"}`,
			jsonBody:     true,
		},
		{
			name:         "instance zone",
			path:         "/istio-test/metadata/instance-zone",
			expectedCode: http.StatusOK,
			expectedBody: `{"instance-zone":"us-central1-a"}`,
			jsonBody:     true,
		},
		{
			name:         "unknown metadata type",
			path:         "/istio-test/metadata/unknown",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Unknown metadata type\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(server.URL + tt.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.jsonBody {
				assert.JSONEq(t, tt.expectedBody, string(body))
				return
			}

			assert.Equal(t, tt.expectedBody, string(body))
		})
	}
}

func TestEnhancedHealthCheckHandler(t *testing.T) {
	t.Run("basic health check response structure", func(t *testing.T) {
		mockClient := NewClient(1*time.Second, 1, 50*time.Millisecond, 500*time.Millisecond, 2.0)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler := EnhancedHealthCheckHandler(mockClient)
		handler(w, req)

		resp := w.Result()
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var healthResp HealthResponse
		err := json.NewDecoder(resp.Body).Decode(&healthResp)
		assert.NoError(t, err)

		assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusDegraded, HealthStatusUnhealthy}, healthResp.Status)
		assert.Contains(t, healthResp.Checks, "metadata_service")
		assert.Contains(t, healthResp.Checks, "http_server")
		assert.NotEmpty(t, healthResp.Uptime)
		assert.Equal(t, "dev", healthResp.Version)
		assert.NotZero(t, healthResp.Timestamp)
		assert.Equal(t, HealthStatusHealthy, healthResp.Checks["http_server"].Status)
		assert.Contains(t, healthResp.Checks["http_server"].Message, "responding")
	})
}

func TestGetVersion(t *testing.T) {
	t.Run("default version", func(t *testing.T) {
		originalVersion := version
		version = "dev"
		defer func() { version = originalVersion }()

		assert.Equal(t, "dev", getVersion())
	})

	t.Run("custom version", func(t *testing.T) {
		originalVersion := version
		version = "v1.2.3"
		defer func() { version = originalVersion }()

		assert.Equal(t, "v1.2.3", getVersion())
	})
}

func TestDetermineOverallHealth(t *testing.T) {
	tests := []struct {
		name     string
		checks   map[string]HealthCheck
		expected HealthStatus
	}{
		{
			name: "all healthy",
			checks: map[string]HealthCheck{
				"service1": {Status: HealthStatusHealthy},
				"service2": {Status: HealthStatusHealthy},
			},
			expected: HealthStatusHealthy,
		},
		{
			name: "one degraded",
			checks: map[string]HealthCheck{
				"service1": {Status: HealthStatusHealthy},
				"service2": {Status: HealthStatusDegraded},
			},
			expected: HealthStatusDegraded,
		},
		{
			name: "one unhealthy",
			checks: map[string]HealthCheck{
				"service1": {Status: HealthStatusHealthy},
				"service2": {Status: HealthStatusUnhealthy},
			},
			expected: HealthStatusUnhealthy,
		},
		{
			name: "mixed with unhealthy",
			checks: map[string]HealthCheck{
				"service1": {Status: HealthStatusHealthy},
				"service2": {Status: HealthStatusDegraded},
				"service3": {Status: HealthStatusUnhealthy},
			},
			expected: HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, determineOverallHealth(tt.checks))
		})
	}
}
