// Package api embeds the OpenAPI document, so the server validates requests
// against the very file the Swift client is generated from.
package api

import _ "embed"

// OpenAPI is api/openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte
