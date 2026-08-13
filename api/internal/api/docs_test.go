package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

/*
 * A hand-written specification is only safe if it cannot drift from the routes
 * the server actually serves.
 *
 * This compares the two in both directions, so adding an endpoint without
 * documenting it fails here rather than being discovered by whoever tries to
 * use the API, and a path that survives in the specification after the route is
 * gone fails too.
 */
func TestSpecAndRoutesAgree(t *testing.T) {
	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(openAPISpec, &spec); err != nil {
		t.Fatalf("the specification is not valid YAML: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("the specification declares no paths")
	}

	documented := map[string]bool{}
	for path, operations := range spec.Paths {
		for method := range operations {
			switch strings.ToUpper(method) {
			case "GET", "POST", "PUT", "DELETE", "PATCH":
				// The specification's paths are relative to the /api server.
				documented[strings.ToUpper(method)+" "+shape("/api"+path)] = true
			}
		}
	}

	served := map[string]bool{}
	for _, route := range NewServer(nil, slog.Default()).Routes() {
		method, path, _ := strings.Cut(route, " ")
		if describesItself(path) {
			continue
		}
		served[method+" "+shape(path)] = true
	}

	for route := range served {
		if !documented[route] {
			t.Errorf("served but not documented: %s", route)
		}
	}
	for route := range documented {
		if !served[route] {
			t.Errorf("documented but not served: %s", route)
		}
	}
}

// The specification, the page that renders it, and the health check describe or
// support the API rather than being part of it.
func describesItself(path string) bool {
	return path == "/docs" || path == "/openapi.yaml" || path == "/healthz"
}

// Parameter names are an implementation detail on both sides, so the comparison
// is between shapes rather than spellings.
func shape(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, "{") {
			segments[i] = "{}"
		}
	}
	return strings.Join(segments, "/")
}

func TestSpecIsServedAndTheDocsPageLoads(t *testing.T) {
	srv := NewServer(nil, slog.Default())

	for _, path := range []string{"/openapi.yaml", "/docs"} {
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("%s returned %d", path, rec.Code)
		}
		if body, _ := io.ReadAll(rec.Body); len(body) == 0 {
			t.Errorf("%s returned an empty body", path)
		}
	}
}

// Loading a third-party script without an integrity hash means a compromised
// CDN can run anything it likes on a page served from this origin.
func TestTheDocsPagePinsItsAssets(t *testing.T) {
	if strings.Contains(swaggerPage, "swagger-ui-dist@5/") {
		t.Error("the assets track a floating major version, which cannot be integrity checked")
	}
	if got := strings.Count(swaggerPage, `integrity="sha384-`); got != 2 {
		t.Errorf("both the stylesheet and the script need an integrity hash, found %d", got)
	}
	if got := strings.Count(swaggerPage, `crossorigin="anonymous"`); got != 2 {
		t.Errorf("integrity hashes are ignored without crossorigin, found %d", got)
	}
}
