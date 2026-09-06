package resolver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPublishedFundingSnapshotMatchesNormalization(t *testing.T) {
	manifestRaw, err := os.ReadFile("../../resolver/publications/funding.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SourceCommit string `json:"source_commit"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceCommit != "5690a33a3cc608320e40872b75a1686e6660db92" {
		t.Fatalf("published Funding snapshot is not pinned to normalization head: %s", manifest.SourceCommit)
	}

	ontologyRaw, err := os.ReadFile("../../representations/funding.owl.ttl")
	if err != nil {
		t.Fatal(err)
	}
	ontology := string(ontologyRaw)
	for _, required := range []string{
		`owl:versionInfo "0.2.0-pre1"`,
		`owl:imports <https://id.exergism.org/ontology/commons>`,
		`<https://id.exergism.org/ontology/governance>`,
		`ecf:FundingAcceptanceDecision a owl:Class ; rdfs:subClassOf ecg:GovernanceDecision`,
	} {
		if !strings.Contains(ontology, required) {
			t.Errorf("published Funding ontology missing normalized assertion %q", required)
		}
	}
	for _, forbidden := range []string{
		"ecf:GovernanceRecord a owl:Class",
		"ecf:GovernanceDecision a owl:Class",
		"ecf:Vote a owl:Class",
		"ecf:ConflictDisclosure a owl:Class",
		"ecf:Person a owl:Class",
		"ecf:membershipEconomicShare",
	} {
		if strings.Contains(ontology, forbidden) {
			t.Errorf("published Funding ontology reintroduces removed term %q", forbidden)
		}
	}

	contextRaw, err := os.ReadFile("../../representations/funding-context.jsonld")
	if err != nil {
		t.Fatal(err)
	}
	var contextDoc struct {
		Context map[string]any `json:"@context"`
	}
	if err := json.Unmarshal(contextRaw, &contextDoc); err != nil {
		t.Fatal(err)
	}
	if got := contextDoc.Context["GovernanceDecision"]; got != "ecg:GovernanceDecision" {
		t.Fatalf("GovernanceDecision must expand to governance#, got %#v", got)
	}
	if _, exists := contextDoc.Context["membershipEconomicShare"]; exists {
		t.Fatal("published Funding context must not expose membershipEconomicShare")
	}
}
