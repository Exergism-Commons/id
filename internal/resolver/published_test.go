package resolver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedRepresentationPathIsDereferenceable(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(name, content string) {
		t.Helper()
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("ontology.html", "<html>ontology</html>")
	mustWrite("ontology.ttl", "@prefix owl: <http://www.w3.org/2002/07/owl#> .")
	registry := `{
      "authority": "https://id.exergism.org/",
      "routes": [{
        "path": "/ontology/test",
        "canonical": "https://id.exergism.org/ontology/test",
        "cache_control": "public, max-age=300",
        "representations": [
          {"media_type":"text/html; charset=utf-8","file":"ontology.html","default":true},
          {"media_type":"text/turtle; charset=utf-8","file":"ontology.ttl","public_path":"/representations/ontology.ttl"}
        ]
      }]
    }`
	registryPath := filepath.Join(dir, "registry.json")
	mustWrite("registry.json", registry)

	h, err := LoadPublished(dir, registryPath)
	if err != nil {
		t.Fatalf("LoadPublished() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/representations/ontology.ttl", nil)
	req.Header.Set("Accept", "text/turtle")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/turtle; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rr.Header().Get("Content-Location"); got != "/representations/ontology.ttl" {
		t.Fatalf("Content-Location = %q", got)
	}
	if link := rr.Header().Get("Link"); !strings.Contains(link, `<https://id.exergism.org/ontology/test>; rel="canonical"`) {
		t.Fatalf("Link = %q, missing canonical ontology IRI", link)
	}
	if !strings.Contains(rr.Body.String(), "@prefix") {
		t.Fatalf("unexpected body %q", rr.Body.String())
	}
}
