package resolver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type catalogNamespaceContract struct {
	ID                string   `json:"id"`
	Namespace         string   `json:"namespace"`
	NamespaceDocument string   `json:"namespace_document"`
	Ontology          string   `json:"ontology"`
	OntologyVersion   string   `json:"ontology_version"`
	VersionIRI        string   `json:"version_iri"`
	Status            string   `json:"status"`
	OwnerRepository   string   `json:"owner_repository"`
	SourceCommit      string   `json:"source_commit"`
	SourcePath        string   `json:"source_path"`
	Imports           []string `json:"imports"`
	TermCatalogStatus string   `json:"term_catalog_status"`
}

type generatedTermNamespace struct {
	OwnerRepository string `json:"owner_repository"`
	SourcePath      string `json:"source_path"`
	SourceCommit    string `json:"source_commit"`
	OntologyVersion string `json:"ontology_version"`
	Terms           []struct {
		IRI       string `json:"iri"`
		LocalName string `json:"local_name"`
		RDFType   string `json:"rdf_type"`
		Status    string `json:"status"`
	} `json:"terms"`
}

func TestAdoptedNamespaceCatalogContractIsExecutable(t *testing.T) {
	h, err := Load("../..", "../../resolver/registry.json")
	if err != nil {
		t.Fatal(err)
	}

	namespaceRaw, err := os.ReadFile("../../catalog/namespaces.json")
	if err != nil {
		t.Fatal(err)
	}
	var namespaceDoc struct {
		Namespaces []catalogNamespaceContract `json:"namespaces"`
	}
	if err := json.Unmarshal(namespaceRaw, &namespaceDoc); err != nil {
		t.Fatal(err)
	}

	termsRaw, err := os.ReadFile("../../catalog/terms.json")
	if err != nil {
		t.Fatal(err)
	}
	var termsDoc struct {
		Generated  bool                              `json:"generated"`
		Namespaces map[string]generatedTermNamespace `json:"namespaces"`
	}
	if err := json.Unmarshal(termsRaw, &termsDoc); err != nil {
		t.Fatal(err)
	}
	if !termsDoc.Generated {
		t.Fatal("catalog/terms.json must be an explicitly generated projection")
	}

	globalIRIs := map[string]string{}
	for _, namespace := range namespaceDoc.Namespaces {
		if namespace.Status != "adopted" {
			continue
		}
		if namespace.Namespace == "" || namespace.NamespaceDocument == "" || namespace.Ontology == "" ||
			namespace.OntologyVersion == "" || namespace.VersionIRI == "" || namespace.OwnerRepository == "" ||
			namespace.SourceCommit == "" || namespace.SourcePath == "" {
			t.Fatalf("adopted namespace %s is missing required owner/source/version metadata", namespace.ID)
		}
		if namespace.TermCatalogStatus != "indexed" {
			t.Fatalf("adopted namespace %s must have term_catalog_status=indexed", namespace.ID)
		}
		if namespace.VersionIRI != namespace.Ontology+"/"+namespace.OntologyVersion {
			t.Fatalf("adopted namespace %s version IRI %q is inconsistent with ontology/version", namespace.ID, namespace.VersionIRI)
		}

		versionPath := strings.TrimPrefix(namespace.VersionIRI, "https://id.exergism.org")
		route, ok := h.routes[versionPath]
		if !ok {
			t.Fatalf("adopted namespace %s version IRI is not registered: %s", namespace.ID, namespace.VersionIRI)
		}
		if !strings.Contains(strings.ToLower(route.CacheControl), "immutable") {
			t.Fatalf("adopted namespace %s version IRI must be immutable", namespace.ID)
		}

		termNamespace, ok := termsDoc.Namespaces[namespace.ID]
		if !ok {
			t.Fatalf("adopted namespace %s is absent from generated term catalog", namespace.ID)
		}
		if termNamespace.OwnerRepository != namespace.OwnerRepository ||
			termNamespace.SourcePath != namespace.SourcePath ||
			termNamespace.SourceCommit != namespace.SourceCommit ||
			termNamespace.OntologyVersion != namespace.OntologyVersion {
			t.Fatalf("term catalog provenance for %s diverges from namespace registry", namespace.ID)
		}
		if len(termNamespace.Terms) == 0 {
			t.Fatalf("adopted namespace %s has no indexed terms", namespace.ID)
		}

		seenLocal := map[string]bool{}
		for _, term := range termNamespace.Terms {
			if !strings.HasPrefix(term.IRI, namespace.Namespace) {
				t.Fatalf("term %s is outside adopted namespace %s", term.IRI, namespace.Namespace)
			}
			if term.LocalName == "" || term.RDFType == "" || term.Status == "" {
				t.Fatalf("term %s is missing discovery metadata", term.IRI)
			}
			if seenLocal[term.LocalName] {
				t.Fatalf("duplicate local name %s in namespace %s", term.LocalName, namespace.ID)
			}
			seenLocal[term.LocalName] = true
			if owner, exists := globalIRIs[term.IRI]; exists {
				t.Fatalf("term IRI %s is indexed by both %s and %s", term.IRI, owner, namespace.ID)
			}
			globalIRIs[term.IRI] = namespace.ID
		}
	}
}
