package internal

import (
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

type GeoIPMiddleware struct {
	reader         *geoip2.Reader
	logger         *slog.Logger
	next           http.Handler
	allowCountries []string
	blockCountries []string
}

func NewGeoIPMiddleware(reader *geoip2.Reader, logger *slog.Logger, next http.Handler, allowCountries, blockCountries []string) *GeoIPMiddleware {
	return &GeoIPMiddleware{
		reader:         reader,
		logger:         logger,
		next:           next,
		allowCountries: allowCountries,
		blockCountries: blockCountries,
	}
}

func (m *GeoIPMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract IP address from request
	remoteAddr := r.Header.Get("X-Forwarded-For")
	if remoteAddr == "" {
		remoteAddr = r.RemoteAddr
	}

	// Parse IP address (remove port if present)
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr // Assume no port was present
	}

	ip := net.ParseIP(host)
	if ip != nil {
		// Always allow localhost and internal IP ranges
		if isLocalOrInternalIP(ip) {
			m.next.ServeHTTP(w, r)
			return
		}

		// Look up country information
		country, err := m.reader.Country(ip)
		if err == nil {
			countryCode := country.Country.IsoCode

			// Check country filtering rules
			if len(m.allowCountries) > 0 {
				// If allow list is configured, only allow requests from those countries
				allowed := false
				for _, allowedCountry := range m.allowCountries {
					if strings.EqualFold(countryCode, allowedCountry) {
						allowed = true
						break
					}
				}
				if !allowed {
					m.logger.Info("Request blocked - country not in allow list",
						"country", countryCode, "ip", host, "allowed_countries", m.allowCountries)
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			} else if len(m.blockCountries) > 0 {
				// If block list is configured, block requests from those countries
				for _, blockedCountry := range m.blockCountries {
					if strings.EqualFold(countryCode, blockedCountry) {
						m.logger.Info("Request blocked - country in block list",
							"country", countryCode, "ip", host, "blocked_countries", m.blockCountries)
						http.Error(w, "Access denied", http.StatusForbidden)
						return
					}
				}
			}

			// Add GeoIP information to request context via headers
			// This allows downstream middleware to access the information
			if countryCode != "" {
				r.Header.Set("X-GeoIP-Country", countryCode)
			}
		}
	}
	m.next.ServeHTTP(w, r)
}

func (m *GeoIPMiddleware) Close() error {
	if m.reader != nil {
		return m.reader.Close()
	}
	return nil
}

// Helper function to find GeoIP2 database file
func FindGeoIP2Database() string {
	// Common paths where GeoIP2 databases might be located
	possiblePaths := []string{
		"./GeoLite2-Country.mmdb",
		"./data/GeoLite2-Country.mmdb",
		"./storage/GeoLite2-Country.mmdb",
		"./fixtures/GeoLite2-Country.mmdb", // <-- This one is for testing.
	}

	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if absPath != "" {
				return absPath
			}
		}
	}

	return ""
}

// isLocalOrInternalIP checks if an IP address is localhost or from internal/private ranges
func isLocalOrInternalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for IPv4 localhost (127.0.0.0/8)
	if ip.IsLoopback() {
		return true
	}

	// Check for IPv6 localhost (::1)
	if ip.Equal(net.IPv6loopback) {
		return true
	}

	// Check for private IPv4 ranges
	// 10.0.0.0/8
	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ipv4[0] == 192 && ipv4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ipv4[0] == 169 && ipv4[1] == 254 {
			return true
		}
	}

	// Check for private IPv6 ranges
	// fc00::/7 (unique local addresses)
	if len(ip) == 16 && (ip[0]&0xfe) == 0xfc {
		return true
	}

	// fe80::/10 (link-local)
	if len(ip) == 16 && ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
		return true
	}

	return false
}
