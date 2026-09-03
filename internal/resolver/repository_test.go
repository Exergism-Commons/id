package resolver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type publicationManifest struct {
	VocabularyNamespace string   `json:"vocabulary_namespace"`
	OntologyIRI         string   `json:"ontology_iri"`
	Records             []string `json:"records"`
}

func TestRepositoryRegistryLoadsAllRegisteredRepresentations(t *testing.T) {
	if _, err := Load("../..", "../../resolver/registry.json"); err != nil {
		t.Fatalf("repository registry must load with every registered representation present: %v", err)
	}
}

func TestFundingPublicationManifestIsFullyResolvable(t *testing.T) {
	h, err := Load("../..", "../../resolver/registry.json")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("../../resolver/publications/funding.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest publicationManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode funding publication manifest: %v", err)
	}

	if manifest.VocabularyNamespace != "https://id.exergism.org/funding#" {
		t.Fatalf("unexpected funding vocabulary namespace %q", manifest.VocabularyNamespace)
	}
	if _, ok := h.routes["/funding"]; !ok {
		t.Fatal("funding namespace document is not registered")
	}

	ontologyPath := strings.TrimPrefix(manifest.OntologyIRI, "https://id.exergism.org")
	if _, ok := h.routes[ontologyPath]; !ok {
		t.Fatalf("funding ontology %q is not registered", manifest.OntologyIRI)
	}

	if len(manifest.Records) == 0 {
		t.Fatal("funding publication manifest has no records")
	}
	for _, iri := range manifest.Records {
		path := strings.TrimPrefix(iri, "https://id.exergism.org")
		route, ok := h.routes[path]
		if !ok {
			t.Errorf("published Funding IRI does not resolve: %s", iri)
			continue
		}
		hasHTML := false
		hasJSONLD := false
		for _, rep := range route.Representations {
			if strings.HasPrefix(rep.MediaType, "text/html") {
				hasHTML = true
			}
			if rep.MediaType == "application/ld+json" {
				hasJSONLD = true
			}
		}
		if !hasHTML || !hasJSONLD {
			t.Errorf("published Funding IRI %s must have human HTML and machine JSON-LD representations", iri)
		}
	}
}
