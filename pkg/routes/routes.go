// Task Board is a self-hosted Kanban application.
// Copyright 2026-present Task Board contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// @title taskboard API
// @description This is the documentation for the [taskboard](https://taskboard.io) API. taskboard is a cross-platform To-do-application with a lot of features, such as sharing projects with users or teams. <!-- ReDoc-Inject: <security-definitions> -->

// @description # Pagination
// @description Every endpoint capable of pagination will return two headers:
// @description * `x-pagination-total-pages`: The total number of available pages for this request
// @description * `x-pagination-result-count`: The number of items returned for this request.
// @description # Permissions
// @description All endpoints which return a single item (project, task, etc.) - no array - will also return a `x-max-permission` header with the max permission the user has on this item as an int where `0` is `Read Only`, `1` is `Read & Write` and `2` is `Admin`.
// @description This can be used to show or hide ui elements based on the permissions the user has.
// @description # Errors
// @description All errors have an error code and a human-readable error message in addition to the http status code. You should always check for the status code in the response, not only the http status code.
// @description Due to limitations in the swagger library we're using for this document, only one error per http status code is documented here. Make sure to check the [error docs](https://taskboard.io/docs/errors/) in taskboard's documentation for a full list of available error codes.
// @description # Authorization
// @description **JWT-Auth:** Main authorization method, used for most of the requests. Needs an `Authorization: Bearer <jwt-token>` header to authenticate successfully.
// @description <!-- ReDoc-Inject: <security-definitions> -->
// @BasePath /api/v1

// @license.url https://github.com/FerrPOINT/task-bord/src/branch/main/LICENSE
// @license.name AGPL-3.0-or-later

// @contact.url https://taskboard.io/contact/
// @contact.name General taskboard contact
// @contact.email hello@taskboard.io

// @securityDefinitions.apikey JWTKeyAuth
// @in header
// @name Authorization

package routes

import (
	"context"
	"log/slog"
	"net"
	"strings"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/log"
	apiv1 "github.com/FerrPOINT/task-bord/pkg/routes/api/v1"
	vmiddleware "github.com/FerrPOINT/task-bord/pkg/routes/middleware"
	ws "github.com/FerrPOINT/task-bord/pkg/websocket"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// matchCORSOrigin checks if an origin matches any of the allowed origin patterns.
// It supports wildcards in the port position (e.g., "http://127.0.0.1:*").
func matchCORSOrigin(origin string, allowedOrigins []string) (string, bool, error) {
	for _, pattern := range allowedOrigins {
		// Exact match
		if origin == pattern {
			return origin, true, nil
		}
		// Allow all
		if pattern == "*" {
			return origin, true, nil
		}
		// Handle wildcard port patterns like "http://127.0.0.1:*" or "http://localhost:*"
		if strings.HasSuffix(pattern, ":*") {
			prefix := strings.TrimSuffix(pattern, ":*")
			// Check if the origin starts with the prefix and has a port after
			if strings.HasPrefix(origin, prefix+":") {
				return origin, true, nil
			}
			// Also match if origin has no port but pattern allows any port
			if origin == prefix {
				return origin, true, nil
			}
		}
	}
	return "", false, nil
}

// NewEcho registers a new Echo instance
func NewEcho() *echo.Echo {
	// Configure Echo with a router that unescapes path parameters.
	// This is needed because Echo v5 does not unescape path params by default.
	// Without this, path parameters like usernames with spaces or apostrophes
	// would remain URL-encoded (e.g., "John%20D%27Urso" instead of "John D'Urso").
	e := echo.NewWithConfig(echo.Config{
		Router: echo.NewRouter(echo.RouterConfig{
			UnescapePathParamValues: true,
		}),
	})

	// Configure IP extraction to prevent rate limit bypass via spoofed headers.
	// Echo's default RealIP() trusts X-Forwarded-For and X-Real-IP unconditionally,
	// which allows attackers to bypass IP-based rate limits.
	// See: https://echo.labstack.com/docs/ip-address
	switch config.ServiceIPExtractionMethod.GetString() {
	case "xff":
		trustOptions := parseTrustedProxies(config.ServiceTrustedProxies.GetString())
		e.IPExtractor = echo.ExtractIPFromXFFHeader(trustOptions...)
		log.Debugf("IP extraction: X-Forwarded-For with %d trusted proxy ranges", len(trustOptions))
	case "realip":
		trustOptions := parseTrustedProxies(config.ServiceTrustedProxies.GetString())
		e.IPExtractor = echo.ExtractIPFromRealIPHeader(trustOptions...)
		log.Debugf("IP extraction: X-Real-IP with %d trusted proxy ranges", len(trustOptions))
	default:
		e.IPExtractor = echo.ExtractIPDirect()
		log.Debugf("IP extraction: direct (TCP remote address)")
	}

	e.Logger = log.NewEchoLogger(config.LogEnabled.GetBool(), config.LogHTTP.GetString(), config.LogFormat.GetString())

	// First middleware in the chain so every request has an ID — reuses the
	// X-Request-Id header from a proxy or generates one — and everything
	// downstream (logging, audit) sees the same value.
	e.Use(middleware.RequestID())

	// Logger
	if config.LogEnabled.GetBool() && config.LogHTTP.GetString() != "off" {
		httpLogger := log.NewHTTPLogger(config.LogEnabled.GetBool(), config.LogHTTP.GetString(), config.LogFormat.GetString())
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogStatus:    true,
			LogURI:       true,
			LogMethod:    true,
			LogLatency:   true,
			LogRemoteIP:  true,
			LogUserAgent: true,
			HandleError:  true,
			LogValuesFunc: func(_ *echo.Context, v middleware.RequestLoggerValues) error {
				if v.Error == nil {
					httpLogger.LogAttrs(context.Background(), slog.LevelInfo, "",
						slog.String("remote_ip", v.RemoteIP),
						slog.String("method", v.Method),
						slog.String("uri", v.URI),
						slog.Int("status", v.Status),
						slog.Duration("latency", v.Latency),
						slog.String("user_agent", v.UserAgent),
					)
				} else {
					httpLogger.LogAttrs(context.Background(), slog.LevelError, "",
						slog.String("remote_ip", v.RemoteIP),
						slog.String("method", v.Method),
						slog.String("uri", v.URI),
						slog.Int("status", v.Status),
						slog.Duration("latency", v.Latency),
						slog.String("user_agent", v.UserAgent),
						slog.String("err", v.Error.Error()),
					)
				}
				return nil
			},
		}))
	}

	// panic recover
	e.Use(middleware.Recover())

	// Normalize PHP-style `foo[]=...` query params to `foo=...` before any
	// handler binds them. Runs globally so /api/v1 benefits.
	e.Use(vmiddleware.NormalizeArrayParams())

	// Validation
	e.Validator = &CustomValidator{}

	// Set body limit to allow file uploads up to the configured size
	// Add some overhead for multipart form data (headers, boundaries, etc.)
	maxFileSize := config.GetMaxFileSizeInMBytes()
	// #nosec G115 - maxFileSize is a configuration value that won't exceed int64 max in practice
	e.Use(middleware.BodyLimit((int64(maxFileSize) + 2) * 1024 * 1024))

	// Set up centralized error handler
	e.HTTPErrorHandler = CreateHTTPErrorHandler(e)

	return e
}

func parseTrustedProxies(proxies string) []echo.TrustOption {
	if proxies == "" {
		return nil
	}

	var options []echo.TrustOption
	for _, cidr := range strings.Split(proxies, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Warningf("Invalid trusted proxy CIDR %q: %v", cidr, err)
			continue
		}
		options = append(options, echo.TrustIPRange(ipNet))
	}
	return options
}

func RegisterRoutes(e *echo.Echo) {


	// healthcheck
	e.GET("/health", HealthcheckHandler)

	setupStaticFrontendFilesHandler(e)

	// CORS
	if config.CorsEnable.GetBool() {
		allowedOrigins := config.CorsOrigins.GetStringSlice()
		log.Infof("CORS enabled with origins: %s", strings.Join(allowedOrigins, ", "))

		// Echo v5 CORS middleware is stricter and doesn't accept wildcards in ports like "http://127.0.0.1:*"
		// We use UnsafeAllowOriginFunc to handle these patterns for backwards compatibility
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{}, // Empty because we use UnsafeAllowOriginFunc
			UnsafeAllowOriginFunc: func(_ *echo.Context, origin string) (string, bool, error) {
				return matchCORSOrigin(origin, allowedOrigins)
			},
			AllowCredentials: true,
			MaxAge:           config.CorsMaxAge.GetInt(),
			Skipper: func(context *echo.Context) bool {
				// CORS only matters for browser-based API calls. There are no
				// non-browser API subsystems (like CalDAV or feed readers) left.
				return false
			},
		}))
	}

	// API Routes
	a2 := e.Group("/api/v1")
	registerAPIRoutesV1(e, a2)
}

// noStoreCacheControl returns middleware that sets `Cache-Control: no-store`
// on all responses. Without this, browsers may heuristically cache JSON
// responses which causes stale data (e.g. newly team-shared projects not
// appearing until a hard refresh). Applied to /api/v1.
func noStoreCacheControl() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", "no-store")
			return next(c)
		}
	}
}

const adminPathPrefix = "/api/v1/admin"

// gateAdminRoutes uses RequireInstanceAdmin as path-scoped middleware.
func gateAdminRoutes() echo.MiddlewareFunc {
	admin := RequireInstanceAdmin()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		gated := admin(next)
		return func(c *echo.Context) error {
			if strings.HasPrefix(c.Request().URL.Path, adminPathPrefix) {
				return gated(c)
			}
			return next(c)
		}
	}
}

// registerAPIRoutesV1 wires the /api/v1 Echo group. JWT middleware is
// attached before any resource handler so Huma's spec and Scalar docs share
// the resource handlers' stack; unauthenticatedAPIPaths keeps public routes open.
func registerAPIRoutesV1(e *echo.Echo, a *echo.Group) {
	a.Use(noStoreCacheControl())
	// Rate limiting and route metrics apply to /api/v1 resource endpoints too.
	setupRateLimit(a, config.RateLimitKind.GetString())
	setupMetricsMiddleware(a)
	// JWT must run before the admin gate: the gate reads the authenticated user.
	a.Use(vmiddleware.JWT())
	// Must come after rate limiting: the gate does a per-request admin DB read,
	// so an unauthenticated flood to /api/v1/admin/* would otherwise be unbounded.
	a.Use(gateAdminRoutes())

	api := apiv1.NewAPI(e, a)

	// WebSockets can't be modeled in OpenAPI and Huma has no WS support, so the
	// upgrade endpoint stays a raw echo route (outside the Huma spec). It
	// authenticates via its first message, so unauthenticatedAPIPaths exempts it
	// from the group's JWT middleware. Health is a Huma op and self-registers via
	// init()/RegisterAll.
	a.GET("/ws", ws.UpgradeHandler)

	// Resources self-register via init(); RegisterAll runs them all + AutoPatch.
	apiv1.RegisterAll(api)
}





