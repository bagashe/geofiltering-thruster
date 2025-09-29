package internal

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/klauspost/compress/gzhttp"
	"github.com/oschwald/geoip2-golang"
)

type HandlerOptions struct {
	badGatewayPage           string
	cache                    Cache
	maxCacheableResponseBody int
	maxRequestBody           int
	targetUrl                *url.URL
	xSendfileEnabled         bool
	gzipCompressionEnabled   bool
	forwardHeaders           bool
	logRequests              bool
	geoIP2Enabled            bool
	allowCountries           []string
	blockCountries           []string
}

func NewHandler(options HandlerOptions) http.Handler {
	handler := NewProxyHandler(options.targetUrl, options.badGatewayPage, options.forwardHeaders)
	handler = NewCacheHandler(options.cache, options.maxCacheableResponseBody, handler)
	handler = NewSendfileHandler(options.xSendfileEnabled, handler)
	handler = NewRequestStartMiddleware(handler)

	if options.gzipCompressionEnabled {
		handler = gzhttp.GzipHandler(handler)
	}

	if options.maxRequestBody > 0 {
		handler = http.MaxBytesHandler(handler, int64(options.maxRequestBody))
	}

	if options.geoIP2Enabled {
		// Find GeoIP2 database automatically
		dbPath := FindGeoIP2Database()
		reader, err := geoip2.Open(dbPath)
		if err != nil {
			slog.Default().Warn("Failed to open GeoIP2 database. NOT loading the GeoIP2 middleware for IP filtering.", "path", dbPath, "error", err)
		} else {
			slog.Default().Info("Loaded GeoIP2 country database & GeoIP2 middleware for IP filtering.")
			handler = NewGeoIPMiddleware(reader, slog.Default(), handler, options.allowCountries, options.blockCountries)
		}
	}

	if options.logRequests {
		handler = NewLoggingMiddleware(slog.Default(), handler)
	}

	return handler
}
