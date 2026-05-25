package openapi_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	OpenAPI    string                            `yaml:"openapi"`
	Paths      map[string]map[string]interface{} `yaml:"paths"`
	Components struct {
		SecuritySchemes map[string]interface{} `yaml:"securitySchemes"`
	} `yaml:"components"`
}

func TestOpenAPIDocumentsServerRoutes(t *testing.T) {
	spec := readOpenAPISpec(t)

	expectedPaths := []string{
		"/health",
		"/ready",
		"/version",
		"/api/v1/items",
		"/api/v1/items/{id}",
		"/api/v1/search",
		"/api/v1/categories",
		"/api/v1/categories/{id}",
		"/api/v1/categories/{id}/groups",
		"/api/v1/groups",
		"/api/v1/groups/{id}",
		"/api/v1/diff",
		"/api/v1/changelog",
		"/api/esi/types/{id}",
		"/api/esi/markets/prices",
		"/api/esi/markets/{regionID}/history/{typeID}",
		"/api/esi/cache/clear",
		"/api/admin/stats",
		"/api/admin/keys",
		"/api/admin/keys/{id}",
		"/api/admin/sde/update",
		"/api/admin/sde/status",
	}

	for _, path := range expectedPaths {
		if _, ok := spec.Paths[path]; !ok {
			t.Fatalf("OpenAPI spec is missing path %s", path)
		}
	}
}

func TestOpenAPIDocumentsSecuritySchemes(t *testing.T) {
	spec := readOpenAPISpec(t)

	for _, scheme := range []string{"ApiKeyAuth", "AdminBasicAuth"} {
		if _, ok := spec.Components.SecuritySchemes[scheme]; !ok {
			t.Fatalf("OpenAPI spec is missing security scheme %s", scheme)
		}
	}

	cacheClear, ok := spec.Paths["/api/esi/cache/clear"]["post"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec is missing POST operation for /api/esi/cache/clear")
	}
	if _, ok := cacheClear["security"]; !ok {
		t.Fatal("OpenAPI POST /api/esi/cache/clear must document API-key security")
	}

	adminKeysPost, ok := spec.Paths["/api/admin/keys"]["post"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec is missing POST operation for /api/admin/keys")
	}
	if _, ok := adminKeysPost["security"]; !ok {
		t.Fatal("OpenAPI POST /api/admin/keys must document admin security")
	}
}

func readOpenAPISpec(t *testing.T) openAPISpec {
	t.Helper()

	path := filepath.Join("..", "..", "..", "api", "openapi.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI spec: %v", err)
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse OpenAPI spec: %v", err)
	}
	if spec.OpenAPI == "" {
		t.Fatal("OpenAPI spec is missing openapi version")
	}

	return spec
}
