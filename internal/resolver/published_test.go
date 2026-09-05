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

func TestFrontendAssetsAreServed(t *testing.T) {
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

	mustWrite("index.html", "<html></html>")
	mustWrite("assets/id.css", "body { max-width: 70rem; }")
	mustWrite("assets/funding-vocabulary.js", "document.documentElement.dataset.ready = '1';")
	registry := `{
      "authority": "https://id.exergism.org/",
      "routes": [{
        "path": "/",
        "canonical": "https://id.exergism.org/",
        "representations": [
          {"media_type":"text/html; charset=utf-8","file":"index.html","default":true}
        ]
      }]
    }`
	registryPath := filepath.Join(dir, "registry.json")
	mustWrite("registry.json", registry)

	h, err := LoadPublished(dir, registryPath)
	if err != nil {
		t.Fatalf("LoadPublished() error = %v", err)
	}

	tests := []struct {
		path        string
		wantType    string
		wantContent string
	}{
		{"/assets/id.css", "text/css", "max-width"},
		{"/assets/funding-vocabulary.js", "javascript", "dataset.ready"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org"+tc.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			if got := rr.Header().Get("Content-Type"); !strings.Contains(got, tc.wantType) {
				t.Fatalf("Content-Type = %q, want substring %q", got, tc.wantType)
			}
			if !strings.Contains(rr.Body.String(), tc.wantContent) {
				t.Fatalf("body = %q, want substring %q", rr.Body.String(), tc.wantContent)
			}
			if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q", got)
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/assets/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("asset directory status = %d, want 404", rr.Code)
	}
}
