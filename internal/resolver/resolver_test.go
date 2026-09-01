package resolver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
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

	mustWrite("index.html", "<html>root</html>")
	mustWrite("root.jsonld", `{"@id":"https://id.exergism.org/"}`)
	mustWrite("exergism/index.html", "<html>exergism</html>")
	mustWrite("ontology.ttl", "@prefix owl: <http://www.w3.org/2002/07/owl#> .")

	registry := `{
      "authority": "https://id.exergism.org/",
      "routes": [
        {
          "path": "/",
          "canonical": "https://id.exergism.org/",
          "representations": [
            {"media_type":"text/html; charset=utf-8","file":"index.html","default":true},
            {"media_type":"application/ld+json","file":"root.jsonld","public_path":"/root.jsonld"}
          ]
        },
        {
          "path": "/exergism",
          "aliases": ["/exergism/"],
          "canonical": "https://id.exergism.org/exergism",
          "representations": [
            {"media_type":"text/html; charset=utf-8","file":"exergism/index.html","default":true}
          ]
        },
        {
          "path": "/ontology/test",
          "canonical": "https://id.exergism.org/ontology/test",
          "representations": [
            {"media_type":"text/html; charset=utf-8","file":"index.html","default":true},
            {"media_type":"text/turtle; charset=utf-8","file":"ontology.ttl","public_path":"/ontology.ttl"}
          ]
        }
      ]
    }`
	registryPath := filepath.Join(dir, "registry.json")
	mustWrite("registry.json", registry)

	h, err := Load(dir, registryPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return h
}

func TestExactHashNamespaceDocumentPathDoesNotRedirect(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/exergism", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "" {
		t.Fatalf("unexpected redirect to %q", location)
	}
}

func TestTrailingSlashCanonicalizesToExactPath(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/exergism/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want 308", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/exergism" {
		t.Fatalf("Location = %q, want /exergism", got)
	}
}

func TestContentNegotiation(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/ontology/test", nil)
	req.Header.Set("Accept", "text/html;q=0.2, text/turtle;q=0.9")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/turtle; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if !strings.Contains(rr.Body.String(), "@prefix") {
		t.Fatalf("unexpected body %q", rr.Body.String())
	}
	if got := rr.Header().Get("Vary"); got != "Accept" {
		t.Fatalf("Vary = %q, want Accept", got)
	}
}

func TestUnsupportedRepresentationReturns406(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/exergism", nil)
	req.Header.Set("Accept", "text/turtle")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotAcceptable {
		t.Fatalf("status = %d, want 406", rr.Code)
	}
}

func TestHeadReturnsHeadersWithoutBody(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodHead, "https://id.exergism.org/", nil)
	req.Header.Set("Accept", "application/ld+json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("HEAD returned body %q", rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/ld+json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestConditionalGetWithETag(t *testing.T) {
	h := testHandler(t)
	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/", nil)
	h.ServeHTTP(first, req)
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	second := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "https://id.exergism.org/", nil)
	req2.Header.Set("If-None-Match", etag)
	h.ServeHTTP(second, req2)
	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", second.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "https://id.exergism.org/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
	if got := rr.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q", got)
	}
}
