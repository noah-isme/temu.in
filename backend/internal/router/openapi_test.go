package router

import (
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"temu.in/backend/internal/config"
	"temu.in/backend/openapi"
)

var specMethods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true, "head": true, "options": true}

// ginPathToOpenAPI converts gin's `:param` / `*param` segments to `{param}`.
func ginPathToOpenAPI(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "*") {
			segments[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segments, "/")
}

func routerOperations(t *testing.T) map[string]bool {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(engine, &config.Config{JWTSecret: "test"}, nil, nil).RegisterAll()
	ops := map[string]bool{}
	for _, r := range engine.Routes() {
		ops[r.Method+" "+ginPathToOpenAPI(r.Path)] = true
	}
	return ops
}

type specOperation struct {
	OperationID string `yaml:"operationId"`
}

// specOperations parses openapi.Spec and returns "METHOD path" -> operationId.
func specOperations(t *testing.T, spec []byte) map[string]string {
	t.Helper()
	var doc struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	ops := map[string]string{}
	for path, item := range doc.Paths {
		for method, node := range item {
			if !specMethods[method] {
				continue // path-level parameters, summary, ...
			}
			var op specOperation
			if err := node.Decode(&op); err != nil {
				t.Fatalf("decode %s %s: %v", method, path, err)
			}
			ops[strings.ToUpper(method)+" "+path] = op.OperationID
		}
	}
	return ops
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// diffOperations returns the operations missing on each side.
func diffOperations(router map[string]bool, spec map[string]string) (missingFromSpec, missingFromRouter []string) {
	for _, op := range sortedKeys(router) {
		if _, ok := spec[op]; !ok {
			missingFromSpec = append(missingFromSpec, op)
		}
	}
	for _, op := range sortedKeys(spec) {
		if !router[op] {
			missingFromRouter = append(missingFromRouter, op)
		}
	}
	return missingFromSpec, missingFromRouter
}

// TestOpenAPIMatchesRouter requires the spec to list exactly the routes gin
// registers (two-way set equality), so a route can never be added to one side
// only.
func TestOpenAPIMatchesRouter(t *testing.T) {
	missingFromSpec, missingFromRouter := diffOperations(routerOperations(t), specOperations(t, openapi.Spec))
	if len(missingFromSpec) > 0 {
		t.Errorf("missing from spec (registered in gin, absent from backend/openapi/openapi.yaml):\n  %s", strings.Join(missingFromSpec, "\n  "))
	}
	if len(missingFromRouter) > 0 {
		t.Errorf("missing from router (in backend/openapi/openapi.yaml, not registered in gin):\n  %s", strings.Join(missingFromRouter, "\n  "))
	}
}

// TestOpenAPIOperationIDsAreUniqueAndPresent keeps the generated client stable.
func TestOpenAPIOperationIDsAreUniqueAndPresent(t *testing.T) {
	seen := map[string]string{}
	ops := specOperations(t, openapi.Spec)
	for _, op := range sortedKeys(ops) {
		id := ops[op]
		if id == "" {
			t.Errorf("%s has no operationId", op)
			continue
		}
		if prev, dup := seen[id]; dup {
			t.Errorf("operationId %q used by both %s and %s", id, prev, op)
		}
		seen[id] = op
	}
}

// TestOpenAPIRouteWalkDetectsDrift proves the comparison is not vacuous:
// removing one path from the spec must be reported as missing from the spec.
func TestOpenAPIRouteWalkDetectsDrift(t *testing.T) {
	var doc map[string]any
	if err := yaml.Unmarshal(openapi.Spec, &doc); err != nil {
		t.Fatal(err)
	}
	delete(doc["paths"].(map[string]any), "/api/providers/{providerId}/bookings")
	mutated, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	missingFromSpec, missingFromRouter := diffOperations(routerOperations(t), specOperations(t, mutated))
	if len(missingFromSpec) != 1 || missingFromSpec[0] != "GET /api/providers/{providerId}/bookings" || len(missingFromRouter) != 0 {
		t.Fatalf("drift not detected: missingFromSpec=%v missingFromRouter=%v", missingFromSpec, missingFromRouter)
	}
}

func TestGinPathToOpenAPI(t *testing.T) {
	cases := map[string]string{
		"/api/providers/:providerId/services/:id": "/api/providers/{providerId}/services/{id}",
		"/api/favorites/":                         "/api/favorites/",
		"/static/*filepath":                       "/static/{filepath}",
	}
	for in, want := range cases {
		if got := ginPathToOpenAPI(in); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
}
