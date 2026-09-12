// Package api contains types and the server interface generated from api/openapi.yaml.
// Run `go generate ./...` after changing the spec to regenerate server.gen.go.
package api

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
