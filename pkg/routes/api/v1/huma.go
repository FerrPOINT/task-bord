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

// Package apiv1 wires Huma onto the /api/v1 Echo group.
package apiv1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/modules/humaecho5"
	"github.com/FerrPOINT/task-bord/pkg/version"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/labstack/echo/v5"
)

// formURLEncodedContentType is the content type the OAuth token endpoint accepts
// in addition to JSON, per RFC 6749.
const formURLEncodedContentType = "application/x-www-form-urlencoded"

// formURLEncodedFormat lets Huma bind application/x-www-form-urlencoded request
// bodies into the same json-tagged structs it uses for JSON: the form values are
// re-marshaled to JSON and decoded via the standard path. Only string scalars
// are produced, which is all the form-encoded endpoints (OAuth token) need.
var formURLEncodedFormat = huma.Format{
	Marshal: func(io.Writer, any) error {
		// Responses are always JSON; this format is request-body only.
		return huma.ErrUnknownContentType
	},
	Unmarshal: func(data []byte, v any) error {
		values, err := url.ParseQuery(string(data))
		if err != nil {
			return err
		}
		flat := make(map[string]string, len(values))
		for key := range values {
			flat[key] = values.Get(key)
		}
		raw, err := json.Marshal(flat)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, v)
	},
}

// GroupPrefix is the URL prefix the Echo group for /api/v1 is mounted at.
const GroupPrefix = "/api/v1"

// NewAPI mounts Huma on the /api/v1 group. Per-resource Register* calls
// live in sibling files.
func NewAPI(e *echo.Echo, g *echo.Group) huma.API {
	cfg := huma.DefaultConfig("taskboard API", version.Version)
	// Disable automatic $schema links in responses; OpenAPI spec is still
	// served from the configured path if needed.
	cfg.OpenAPIPath = ""
	// Disable Huma's default SchemaLinkTransformer so responses don't include $schema.
	cfg.CreateHooks = nil
	// Huma's built-in docs would load from unpkg.com — we serve Scalar locally instead.
	cfg.DocsPath = ""
	// Compatibility transformer: unwrap Huma envelopes for the Vue frontend.
	cfg.Transformers = append(cfg.Transformers, V1CompatTransformer)
	// Real presence/format rules live in `valid:` tags, enforced by govalidator in
	// the Register wrapper; leave the schema permissive so partial updates work.
	cfg.FieldsOptionalByDefault = true
	// Accept application/x-www-form-urlencoded bodies (the OAuth token endpoint)
	// alongside JSON. Copy the default map so we don't mutate the package global.
	formats := make(map[string]huma.Format, len(cfg.Formats)+1)
	for ct, f := range cfg.Formats {
		formats[ct] = f
	}
	formats[formURLEncodedContentType] = formURLEncodedFormat
	cfg.Formats = formats

	api := humaecho5.NewWithGroup(e, g, GroupPrefix, cfg)
	oapi := api.OpenAPI()
	if oapi.Components.SecuritySchemes == nil {
		oapi.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	// v1 conflated JWTs and tk_-prefixed API tokens under JWTKeyAuth; v2
	// declares them separately so SDK generators and /api/v1/docs distinguish them.
	oapi.Components.SecuritySchemes["JWTKeyAuth"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "User session JWT issued via /api/v1/login.",
	}
	// Applied globally; public endpoints (spec, docs) opt out with an empty Security list.
	oapi.Security = []map[string][]string{
		{"JWTKeyAuth": {}},
	}
	// The relative entry MUST stay at index 0. Huma's SchemaLinkTransformer
	// reads Servers[0] in three places (getAPIPrefix, addSchemaField, the
	// runtime Transform fallback). With a path-bearing absolute URL at
	// index 0, the runtime fallback concatenates that URL onto a ref that
	// already includes /api/v1, producing a double-prefixed $schema link
	// like https://host/api/v1/api/v1/schemas/Label.json. A relative URL
	// at index 0 keeps the prefix in the transformer's bookkeeping while
	// the absolute URL at index 1 advertises the deployment URL to SDK
	// generators and docs UIs.
	servers := []*huma.Server{{URL: GroupPrefix}}
	if publicURL := strings.TrimRight(config.ServicePublicURL.GetString(), "/"); publicURL != "" {
		servers = append(servers, &huma.Server{URL: publicURL + GroupPrefix})
	}
	oapi.Servers = servers
	return api
}

// Register wraps huma.Register with verb-based DefaultStatus: POST → 201,
// DELETE → 204. Anything else (including an explicit op.DefaultStatus) is untouched.
//
// It also runs govalidator before the handler — i.e. before handler.Do*'s
// permission check — so v2 validates-then-authorizes like v1.
func Register[I, O any](api huma.API, op huma.Operation, handler func(context.Context, *I) (*O, error)) {
	if op.DefaultStatus == 0 {
		switch op.Method {
		case http.MethodPost:
			op.DefaultStatus = http.StatusCreated
		case http.MethodDelete:
			op.DefaultStatus = http.StatusNoContent
		}
	}
	wrapped := func(ctx context.Context, in *I) (*O, error) {
		if err := validateInputBody(in); err != nil {
			return nil, translateDomainError(err)
		}
		return handler(ctx, in)
	}
	huma.Register(api, op, wrapped)
}

// EnableAutoPatch synthesises a PATCH for every resource that already
// registered GET + PUT. Must be called AFTER all Register* calls.
func EnableAutoPatch(api huma.API) {
	autopatch.AutoPatch(api)

	// AutoPatch names each synthesised PATCH after the GET operation
	// ("Patch labels-read"), which reads poorly in the docs nav. Rewrite
	// the summary from the sibling PUT so it reads like "Update a label
	// (partial)". Only touch summaries AutoPatch generated (the "Patch "
	// prefix) so a hand-registered PATCH is left alone.
	for _, item := range api.OpenAPI().Paths {
		if item == nil || item.Patch == nil || item.Put == nil {
			continue
		}
		if item.Put.Summary != "" && strings.HasPrefix(item.Patch.Summary, "Patch ") {
			item.Patch.Summary = item.Put.Summary + " (partial)"
		}
	}
}
