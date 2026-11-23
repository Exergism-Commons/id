package resolver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const (
	fundingVersionIRI       = "https://id.exergism.org/ontology/funding/0.2.0-pre1"
	fundingLegacyContextIRI = "https://id.exergism.org/context/funding/0.1.0-draft"
	fundingCurrentContext   = "https://id.exergism.org/context/funding/0.2.0-pre1"
	fundingContextAlias     = "https://id.exergism.org/representations/funding-context.jsonld"
	fundingFrozenRecordIRI  = "https://id.exergism.org/funding/id/ECF-DEC-MRG-BOOTSTRAP-001"
	immutableCacheControl   = "public, max-age=31536000, immutable"
)

type fundingPublication struct {
	SourceCommit       string `json:"source_commit"`
	PublicationVersion string `json:"publication_version"`
	OntologyIRI        string `json:"ontology_iri"`
	OntologyVersionIRI string `json:"ontology_version_iri"`
	Contexts           struct {
		Legacy  string `json:"legacy"`
		Current string `json:"current"`
	} `json:"contexts"`
	Records []string `json:"records"`
}

func loadFundingPublication(t *testing.T) fundingPublication {
	t.Helper()
	raw, err := os.ReadFile("../../resolver/publications/funding.json")
	if err != nil {
		t.Fatal(err)
	}
	var publication fundingPublication
	if err := json.Unmarshal(raw, &publication); err != nil {
		t.Fatalf("decode funding publication manifest: %v", err)
	}
	return publication
}

func TestFundingPublicationDeclaresImmutableVersionAndContexts(t *testing.T) {
	publication := loadFundingPublication(t)
	if publication.PublicationVersion != "0.2.0-pre1" {
		t.Fatalf("unexpected Funding publication version %q", publication.PublicationVersion)
	}
	if publication.OntologyVersionIRI != fundingVersionIRI {
		t.Fatalf("unexpected Funding ontology version IRI %q", publication.OntologyVersionIRI)
	}
	if publication.OntologyVersionIRI != publication.OntologyIRI+"/"+publication.PublicationVersion {
		t.Fatalf("Funding version IRI %q is inconsistent with ontology/version", publication.OntologyVersionIRI)
	}
	if publication.Contexts.Legacy != fundingLegacyContextIRI || publication.Contexts.Current != fundingCurrentContext {
		t.Fatalf("unexpected Funding JSON-LD contexts %+v", publication.Contexts)
	}
}

func TestFundingImmutableRoutesServeExactPublicationBytes(t *testing.T) {
	h, err := LoadPublished("../..", "../../resolver/registry.json")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		iri       string
		mediaType string
		accept    string
		file      string
	}{
		{fundingVersionIRI, "text/turtle; charset=utf-8", "text/turtle", "representations/snapshots/funding-0.2.0-pre1.ttl"},
		{fundingLegacyContextIRI, "application/ld+json", "application/ld+json, application/json;q=0.9", "representations/contexts/funding-context-0.1.0-draft.jsonld"},
		{fundingCurrentContext, "application/ld+json", "application/ld+json, application/json;q=0.9", "representations/contexts/funding-context-0.2.0-pre1.jsonld"},
		{fundingContextAlias, "application/ld+json; charset=utf-8", "application/ld+json", "representations/funding-context.jsonld"},
	}

	for _, tc := range cases {
		t.Run(tc.iri, func(t *testing.T) {
			path := strings.TrimPrefix(tc.iri, "https://id.exergism.org")
			route, ok := h.core.routes[path]
			if !ok {
				t.Fatalf("immutable Funding route %s is not registered", path)
			}
			if route.Canonical != tc.iri {
				t.Fatalf("route %s canonical = %q", path, route.Canonical)
			}
			if route.CacheControl != immutableCacheControl {
				t.Fatalf("route %s must be immutable, got %q", path, route.CacheControl)
			}
			if len(route.Aliases) != 0 {
				t.Fatalf("immutable route %s must not register aliases", path)
			}
			if len(route.Representations) != 1 {
				t.Fatalf("route %s must expose exactly one representation", path)
			}
			rep := route.Representations[0]
			if rep.File != tc.file || rep.MediaType != tc.mediaType || !rep.Default || rep.PublicPath != "" {
				t.Fatalf("route %s has unexpected representation %#v", path, rep)
			}

			want, err := os.ReadFile("../../" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, tc.iri, nil)
			req.Header.Set("Accept", tc.accept)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d", path, rr.Code)
			}
			if got := rr.Header().Get("Content-Type"); got != tc.mediaType {
				t.Fatalf("GET %s Content-Type = %q", path, got)
			}
			if got := rr.Header().Get("Cache-Control"); got != immutableCacheControl {
				t.Fatalf("GET %s Cache-Control = %q", path, got)
			}
			if !bytes.Equal(rr.Body.Bytes(), want) {
				t.Fatalf("GET %s did not return the registered representation bytes", path)
			}
		})
	}
}

func TestFundingLegacyContextAliasIsTheFrozenLegacyContext(t *testing.T) {
	alias, err := os.ReadFile("../../representations/funding-context.jsonld")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := os.ReadFile("../../representations/contexts/funding-context-0.1.0-draft.jsonld")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(alias, legacy) {
		t.Fatal("historical /representations/funding-context.jsonld must stay byte-identical to the immutable 0.1.0-draft context")
	}
}

func TestFundingRecordsUseImmutablePublishedContexts(t *testing.T) {
	h, err := Load("../..", "../../resolver/registry.json")
	if err != nil {
		t.Fatal(err)
	}
	publication := loadFundingPublication(t)
	allowed := map[string]bool{publication.Contexts.Legacy: true, publication.Contexts.Current: true}
	for iri := range allowed {
		route, ok := h.routes[strings.TrimPrefix(iri, "https://id.exergism.org")]
		if !ok || route.CacheControl != immutableCacheControl {
			t.Fatalf("Funding record context %s must be a registered immutable route", iri)
		}
	}

	legacyUsers := 0
	for _, iri := range publication.Records {
		route, ok := h.routes[strings.TrimPrefix(iri, "https://id.exergism.org")]
		if !ok {
			t.Fatalf("published Funding record %s is not registered", iri)
		}
		var file string
		for _, rep := range route.Representations {
			if rep.MediaType == "application/ld+json" {
				file = rep.File
			}
		}
		if file == "" {
			t.Fatalf("published Funding record %s has no JSON-LD representation", iri)
		}
		raw, err := os.ReadFile("../../" + file)
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Context  any    `json:"@context"`
			ID       string `json:"@id"`
			StableID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatalf("decode %s: %v", file, err)
		}
		context, ok := record.Context.(string)
		if !ok || !allowed[context] {
			t.Fatalf("record %s must reference one immutable Funding context IRI, got %#v", iri, record.Context)
		}
		if record.ID != iri || !strings.HasSuffix(iri, "/"+record.StableID) {
			t.Fatalf("record %s has inconsistent @id/id (%q, %q)", iri, record.ID, record.StableID)
		}
		if context == publication.Contexts.Legacy {
			legacyUsers++
			if iri != fundingFrozenRecordIRI {
				t.Fatalf("only %s keeps the frozen 0.1.0-draft interpretation; %s must not", fundingFrozenRecordIRI, iri)
			}
		}
	}
	if legacyUsers != 1 {
		t.Fatalf("%s must keep its frozen 0.1.0-draft interpretation", fundingFrozenRecordIRI)
	}
}

func TestFundingOntologyVersionClosureIsOwnerAuthored(t *testing.T) {
	h, err := Load("../..", "../../resolver/registry.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../representations/snapshots/funding-0.2.0-pre1.ttl")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Count(text, "owl:versionIRI <"+fundingVersionIRI+">") != 1 {
		t.Fatal("Funding must declare its authoritative 0.2.0-pre1 versionIRI exactly once")
	}
	for _, dependency := range []string{
		"https://id.exergism.org/ontology/commons/0.1-PRE2",
		"https://id.exergism.org/ontology/governance/0.1-PRE2",
	} {
		if strings.Count(text, "<"+dependency+">") != 1 {
			t.Fatalf("Funding 0.2.0-pre1 must import %s exactly once", dependency)
		}
		route, ok := h.routes[strings.TrimPrefix(dependency, "https://id.exergism.org")]
		if !ok || route.CacheControl != immutableCacheControl {
			t.Fatalf("imported version IRI %s must be a registered immutable route", dependency)
		}
	}
	for _, mutable := range []string{
		"<https://id.exergism.org/ontology/commons>",
		"<https://id.exergism.org/ontology/governance>",
	} {
		if strings.Contains(text, mutable) {
			t.Fatalf("Funding versioned artifact must not reference mutable current IRI %s", mutable)
		}
	}
}
