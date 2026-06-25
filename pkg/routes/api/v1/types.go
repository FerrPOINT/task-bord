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

package apiv1

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

// Paginated is the standard list-response envelope for every /api/v1 list operation.
type Paginated[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int64 `json:"total_pages"`
}

// NewPaginated builds a Paginated envelope. Nil items become an empty
// slice so the JSON response is [] rather than null.
func NewPaginated[T any](items []T, total int64, page, perPage int) Paginated[T] {
	if items == nil {
		items = []T{}
	}
	var totalPages int64
	if perPage > 0 {
		totalPages = (total + int64(perPage) - 1) / int64(perPage)
	}
	return Paginated[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}

// ListParams carries the standard (page, per_page, q) query shape for list operations.
type ListParams struct {
	Page    int    `query:"page"     default:"1"  minimum:"1" doc:"1-based page number."`
	PerPage int    `query:"per_page" default:"50" minimum:"1" maximum:"1000" doc:"Items per page (max 1000)."`
	Q       string `query:"q" doc:"Search query; filters the list to items matching this string."`
}

// singleBody is the create/update response envelope (no ETag).
type singleBody[T any] struct {
	Body *T
}

// singleReadBody is the read response envelope; carries ETag for If-None-Match.
type singleReadBody[T any] struct {
	ETag string `header:"ETag"`
	Body *T
}

// permission is folded into the ETag so a share/role change invalidates the cache.
func conditionalReadResponse[T any](p *conditional.Params, body *T, modified time.Time, permission int) (*singleReadBody[T], error) {
	e := fmt.Sprintf("%d-%d", modified.UnixNano(), permission)
	if p.HasConditionalParams() {
		if err := p.PreconditionFailed(e, modified); err != nil {
			return nil, err
		}
	}
	return &singleReadBody[T]{ETag: `"` + e + `"`, Body: body}, nil
}

// emptyBody marks delete / no-content operations.
type emptyBody struct{}

// V1CompatTransformer makes Huma responses look like the legacy v1 API the
// Vue frontend expects: list endpoints return a plain array and pagination
// headers, create/update/read endpoints return the bare model, and
// max_permission is exposed as a response header.
var V1CompatTransformer = huma.Transformer(func(ctx huma.Context, status string, data any) (any, error) {
	if data == nil {
		return data, nil
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	// Paginated list: return Items and set pagination headers.
	if strings.HasPrefix(t.String(), "apiv1.Paginated[") {
		total := v.FieldByName("Total").Int()
		page := int(v.FieldByName("Page").Int())
		perPage := int(v.FieldByName("PerPage").Int())
		totalPages := v.FieldByName("TotalPages").Int()
		items := v.FieldByName("Items").Interface()
		ctx.AppendHeader("X-Pagination-Result-Count", fmt.Sprintf("%d", total))
		ctx.AppendHeader("X-Pagination-Total-Pages", fmt.Sprintf("%d", totalPages))
		ctx.AppendHeader("X-Pagination-Current-Page", fmt.Sprintf("%d", page))
		ctx.AppendHeader("X-Pagination-Per-Page", fmt.Sprintf("%d", perPage))
		return items, nil
	}

	// Single-object envelope: unwrap Body.
	if bodyField := v.FieldByName("Body"); bodyField.IsValid() {
		body := bodyField.Interface()
		// Try to pull max_permission from the body or the outer struct.
		if mp := maxPermissionFrom(body); mp >= 0 {
			ctx.AppendHeader("X-Max-Permission", fmt.Sprintf("%d", mp))
		}
		return body, nil
	}

	return data, nil
})

func maxPermissionFrom(v any) int {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return -1
	}
	f := rv.FieldByName("MaxPermission")
	if !f.IsValid() {
		return -1
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(f.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(f.Uint())
	}
	return -1
}
