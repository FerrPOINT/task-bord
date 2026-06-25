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

package middleware

import (
	"net/http"
	"strings"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/modules/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// unauthenticatedAPIPaths holds /api/v1 routes that must be reachable without
// a JWT. It is kept in sync with the public endpoints registered in
// pkg/routes/api/v1 and the raw echo routes mounted on the /api/v1 group.
var unauthenticatedAPIPaths = map[string]bool{
	"/api/v1/openapi.json":              true,
	"/api/v1/openapi.yaml":              true,
	"/api/v1/openapi-3.0.json":          true,
	"/api/v1/openapi-3.0.yaml":          true,
	"/api/v1/docs":                      true,
	"/api/v1/docs/scalar.standalone.js": true,
	"/api/v1/schemas/:schema":           true,
	"/api/v1/info":                      true,
	"/api/v1/health":                    true,

	"/api/v1/register":               true,
	"/api/v1/login":                  true,
	"/api/v1/user/password/token":    true,
	"/api/v1/user/password/reset":    true,
	"/api/v1/user/confirm":          true,
	"/api/v1/user/token/refresh":     true,

	// Testing endpoints authenticate with the testing token via a custom
	// Authorization header, not a JWT; mounted only when that token is set.
	"/api/v1/test/all":    true,
	"/api/v1/test/:table": true,

	// WebSocket upgrade authenticates via its first message.
	"/api/v1/ws": true,
}

// JWT parses and validates the `Authorization: Bearer <jwt>` token for all
// /api/v1 requests except the public paths listed above. On success it stores
// the parsed *jwt.Token under the "user" key in the echo context, which the
// auth package reads via GetAuthFromClaims.
func JWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := c.Path()
			if !strings.HasPrefix(path, "/api/v1") || unauthenticatedAPIPaths[path] {
				return next(c)
			}

			authHeader := c.Request().Header.Get("Authorization")
			const prefix = "bearer "
			if !strings.HasPrefix(strings.ToLower(authHeader), prefix) {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid authorization header")
			}
			tokenString := strings.TrimSpace(authHeader[len(prefix):])

			token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
				return []byte(config.ServiceSecret.GetString()), nil
			})
			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
			}
			typ, ok := claims["type"].(float64)
			if !ok || int(typ) != auth.AuthTypeUser {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token type")
			}

			c.Set("user", token)
			return next(c)
		}
	}
}
