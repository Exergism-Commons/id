package resolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouteCannotBeShadowedByPreviouslyRegisteredAlias(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"first.html", "second.html"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<html></html>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	registry := `{
      "authority":"https://id.exergism.org/",
      "routes":[
        {
          "path":"/first",
          "aliases":["/shadowed"],
          "canonical":"https://id.exergism.org/first",
          "representations":[{"media_type":"text/html","file":"first.html","default":true}]
        },
        {
          "path":"/shadowed",
          "canonical":"https://id.exergism.org/shadowed",
          "representations":[{"media_type":"text/html","file":"second.html","default":true}]
        }
      ]
    }`
	registryPath := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(registryPath, []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir, registryPath)
	if err == nil {
		t.Fatal("Load() accepted a canonical route path shadowed by a prior alias")
	}
	if !strings.Contains(err.Error(), "previously registered alias") {
		t.Fatalf("unexpected error: %v", err)
	}
}
