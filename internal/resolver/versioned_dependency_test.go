package resolver

import (
    "encoding/json"
    "os"
    "testing"
)

type dependencyRoute struct {
    Path          string `json:"path"`
    Canonical     string `json:"canonical"`
    CacheControl  string `json:"cache_control"`
    Representations []struct {
        MediaType string `json:"media_type"`
        File      string `json:"file"`
        Default   bool   `json:"default"`
    } `json:"representations"`
}

func TestGovernanceDependencySnapshotsUseImmutableCommitRoutes(t *testing.T) {
    raw, err := os.ReadFile("../../resolver/registry.json")
    if err != nil {
        t.Fatal(err)
    }
    var registry struct {
        Routes []dependencyRoute `json:"routes"`
    }
    if err := json.Unmarshal(raw, &registry); err != nil {
        t.Fatal(err)
    }

    expected := map[string]string{
        "/ontology/commons/06e614c21f9623658c16175a381279f9c36ef526": "representations/commons.ttl",
        "/ontology/governance/06e614c21f9623658c16175a381279f9c36ef526": "representations/governance.ttl",
    }
    for path, file := range expected {
        var found *dependencyRoute
        for i := range registry.Routes {
            if registry.Routes[i].Path == path {
                found = &registry.Routes[i]
                break
            }
        }
        if found == nil {
            t.Fatalf("immutable dependency route %s is not registered", path)
        }
        if found.Canonical != "https://id.exergism.org"+path {
            t.Fatalf("route %s canonical mismatch: %s", path, found.Canonical)
        }
        if found.CacheControl != "public, max-age=31536000, immutable" {
            t.Fatalf("route %s must be immutable, got %q", path, found.CacheControl)
        }
        if len(found.Representations) != 1 {
            t.Fatalf("route %s must expose exactly one immutable Turtle representation", path)
        }
        representation := found.Representations[0]
        if representation.File != file || representation.MediaType != "text/turtle; charset=utf-8" || !representation.Default {
            t.Fatalf("route %s has unexpected representation: %#v", path, representation)
        }
    }
}
