// Package openapi embeds the hand-written OpenAPI 3.0.3 contract for the API.
//
// The spec is the source of truth: internal/router tests require it to list
// exactly the routes gin registers and validate real responses against it,
// and the frontend generates its TypeScript client from it.
package openapi

import _ "embed"

// Spec is the raw openapi.yaml document, served at GET /api/openapi.yaml.
//
//go:embed openapi.yaml
var Spec []byte
