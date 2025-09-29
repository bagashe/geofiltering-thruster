package internal

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert"
)

func TestGeoIPMiddleware_ServeHTTP(t *testing.T) {
	logger := slog.Default()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if GeoIP headers are present
		country := r.Header.Get("X-GeoIP-Country")

		// Write the values back in response for testing
		if country != "" {
			w.Header().Set("Test-Country", country)
		}

		w.WriteHeader(http.StatusOK)
	})

	dbPath := FindGeoIP2Database()
	reader, _ := geoip2.Open(dbPath)
	middleware := NewGeoIPMiddleware(reader, logger, nextHandler, []string{"US"}, []string{})

	t.Run("handles localhost request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345" // Localhost IP - should bypass GeoIP

		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		// Localhost requests bypass GeoIP, so no headers should be set
		assert.Empty(t, rec.Header().Get("Test-Country"))
	})

	t.Run("handles private network request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345" // Private IP - should bypass GeoIP

		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		// Private network requests bypass GeoIP, so no headers should be set
		assert.Empty(t, rec.Header().Get("Test-Country"))
	})

	t.Run("handles request with X-Forwarded-For header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		req.RemoteAddr = "10.0.0.1:12345" // Internal IP

		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		// Should use X-Forwarded-For IP (localhost) which bypasses GeoIP
		assert.Empty(t, rec.Header().Get("Test-Country"))
	})
}

func TestIsLocalOrInternalIP(t *testing.T) {
	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4 localhost", "127.0.0.1", true},
		{"IPv4 localhost range", "127.1.1.1", true},
		{"IPv6 localhost", "::1", true},
		{"Private 10.x.x.x", "10.0.0.1", true},
		{"Private 192.168.x.x", "192.168.1.1", true},
		{"Private 172.16.x.x", "172.16.0.1", true},
		{"Private 172.31.x.x", "172.31.255.255", true},
		{"Link-local 169.254.x.x", "169.254.1.1", true},
		{"Public IP", "203.0.113.1", false},
		{"Google DNS", "8.8.8.8", false},
		{"Invalid IP", "invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip := parseIP(tc.ip)
			result := isLocalOrInternalIP(ip)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindGeoIP2Database(t *testing.T) {
	// This test just ensures the function doesn't panic
	// In a real environment, it would find actual database files
	result := FindGeoIP2Database()
	// Result could be empty string if no databases found, which is fine
	assert.IsType(t, "", result)
}

// Helper function for testing
func parseIP(s string) net.IP {
	return net.ParseIP(s)
}
