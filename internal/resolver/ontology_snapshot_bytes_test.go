package resolver

import (
    "crypto/sha1"
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "testing"
)

func gitBlobSHA(data []byte) string {
    header := []byte(fmt.Sprintf("blob %d\x00", len(data)))
    sum := sha1.Sum(append(header, data...))
    return fmt.Sprintf("%x", sum)
}

type ontologySnapshotCatalog struct {
    SchemaVersion    int    `json:"schema_version"`
    SourceRepository string `json:"source_repository"`
    SourceCommit     string `json:"source_commit"`
    Snapshots []struct {
        ID                             string `json:"id"`
        SourceRepository               string `json:"source_repository"`
        SourceCommit                   string `json:"source_commit"`
        VersionIRI                     string `json:"version_iri"`
        ProfileIRI                     string `json:"profile_iri"`
        ContextIRI                     string `json:"context_iri"`
        SourcePath                     string `json:"source_path"`
        SourceGitBlobSHA               string `json:"source_git_blob_sha"`
        CurrentPublicationPath         string `json:"current_publication_path"`
        HistoricalAliasPublicationPath string `json:"historical_alias_publication_path"`
        VersionedPublicationPath       string `json:"versioned_publication_path"`
        PublicationGitBlobSHA          string `json:"publication_git_blob_sha"`
        PublicationTransform           string `json:"publication_transform"`
    } `json:"snapshots"`
}

// Owner source commits pinned by the provenance workflows. Repinning an owner
// is a reviewed change to this map, the catalog and the matching workflow.
var pinnedOwnerSourceCommits = map[string]string{
    "Exergism-Commons/governance": "517c0df4c905c79ec2b2824231b40ccaa6a1b510",
    "Exergism-Commons/funding":    "00a11736330d3d8d8f8a492497f470a66e667cdb",
}

func TestOwnerPublicationBytesMatchManifestWithoutTransformation(t *testing.T) {
    h, err := Load("../..", "../../resolver/registry.json")
    if err != nil {
        t.Fatal(err)
    }
    raw, err := os.ReadFile("../../catalog/ontology-snapshots.json")
    if err != nil {
        t.Fatal(err)
    }
    var catalog ontologySnapshotCatalog
    if err := json.Unmarshal(raw, &catalog); err != nil {
        t.Fatal(err)
    }
    if catalog.SchemaVersion != 2 {
        t.Fatalf("unexpected ontology snapshot schema version %d", catalog.SchemaVersion)
    }
    // The top-level owner is the default; snapshots from another owner
    // repository declare their own source_repository and source_commit.
    if catalog.SourceRepository != "Exergism-Commons/governance" {
        t.Fatalf("unexpected default source repository %q", catalog.SourceRepository)
    }
    if catalog.SourceCommit != pinnedOwnerSourceCommits[catalog.SourceRepository] {
        t.Fatalf("unexpected Governance source commit %q", catalog.SourceCommit)
    }

    expected := map[string]string{
        "commons-0.1-PRE2":               "Exergism-Commons/governance",
        "governance-0.1-PRE2":            "Exergism-Commons/governance",
        "governance-authority-0.1-DRAFT": "Exergism-Commons/governance",
        "funding-0.2.0-pre1":             "Exergism-Commons/funding",
        "funding-context-0.1.0-draft":    "Exergism-Commons/funding",
        "funding-context-0.2.0-pre1":     "Exergism-Commons/funding",
    }
    if len(catalog.Snapshots) != len(expected) {
        t.Fatalf("expected %d owner snapshots, got %d", len(expected), len(catalog.Snapshots))
    }

    for _, snapshot := range catalog.Snapshots {
        repository, commit := catalog.SourceRepository, catalog.SourceCommit
        if snapshot.SourceRepository != "" || snapshot.SourceCommit != "" {
            if snapshot.SourceRepository == "" || snapshot.SourceCommit == "" || snapshot.SourceRepository == catalog.SourceRepository {
                t.Fatalf("snapshot %s must override both source_repository and source_commit with another owner", snapshot.ID)
            }
            repository, commit = snapshot.SourceRepository, snapshot.SourceCommit
        }
        owner, ok := expected[snapshot.ID]
        if !ok || repository != owner {
            t.Fatalf("unexpected snapshot %s from %q", snapshot.ID, repository)
        }
        delete(expected, snapshot.ID)
        if commit != pinnedOwnerSourceCommits[owner] {
            t.Fatalf("snapshot %s source commit %q is not the pinned %s commit", snapshot.ID, commit, owner)
        }
        if snapshot.PublicationTransform != "none" {
            t.Fatalf("id must not transform semantic source bytes for %s", snapshot.ID)
        }
        if snapshot.SourceGitBlobSHA != snapshot.PublicationGitBlobSHA {
            t.Fatalf("source/publication blob mismatch for %s", snapshot.ID)
        }
        data, err := os.ReadFile("../../" + snapshot.VersionedPublicationPath)
        if err != nil {
            t.Fatalf("read %s: %v", snapshot.VersionedPublicationPath, err)
        }
        if got := gitBlobSHA(data); got != snapshot.PublicationGitBlobSHA {
            t.Fatalf("versioned publication %s drifted: got %s want %s", snapshot.ID, got, snapshot.PublicationGitBlobSHA)
        }
        for _, path := range []string{snapshot.CurrentPublicationPath, snapshot.HistoricalAliasPublicationPath} {
            if path == "" {
                continue
            }
            current, err := os.ReadFile("../../" + path)
            if err != nil {
                t.Fatalf("read %s: %v", path, err)
            }
            if got := gitBlobSHA(current); got != snapshot.SourceGitBlobSHA {
                t.Fatalf("publication %s of %s is not the adopted source blob: got %s want %s", path, snapshot.ID, got, snapshot.SourceGitBlobSHA)
            }
        }

        identities := 0
        var identity string
        for _, iri := range []string{snapshot.VersionIRI, snapshot.ProfileIRI, snapshot.ContextIRI} {
            if iri != "" {
                identities++
                identity = iri
            }
        }
        if identities != 1 {
            t.Fatalf("snapshot %s must declare exactly one version, profile or context IRI", snapshot.ID)
        }
        route, ok := h.routes[strings.TrimPrefix(identity, "https://id.exergism.org")]
        if !ok {
            t.Fatalf("snapshot %s identity %s is not registered", snapshot.ID, identity)
        }
        if !strings.Contains(strings.ToLower(route.CacheControl), "immutable") ||
            len(route.Representations) != 1 || route.Representations[0].File != snapshot.VersionedPublicationPath {
            t.Fatalf("snapshot %s identity %s must be an immutable route serving only %s", snapshot.ID, identity, snapshot.VersionedPublicationPath)
        }
    }
    if len(expected) != 0 {
        t.Fatalf("missing owner snapshots: %v", expected)
    }
}

func TestGovernanceOntologyVersionClosureIsOwnerAuthored(t *testing.T) {
    commons, err := os.ReadFile("../../representations/snapshots/commons-0.1-PRE2.ttl")
    if err != nil {
        t.Fatal(err)
    }
    governance, err := os.ReadFile("../../representations/snapshots/governance-0.1-PRE2.ttl")
    if err != nil {
        t.Fatal(err)
    }

    commonsVersionIRI := "owl:versionIRI <https://id.exergism.org/ontology/commons/0.1-PRE2>"
    governanceVersionIRI := "owl:versionIRI <https://id.exergism.org/ontology/governance/0.1-PRE2>"
    frozenImport := "owl:imports <https://id.exergism.org/ontology/commons/0.1-PRE2>"

    if strings.Count(string(commons), commonsVersionIRI) != 1 {
        t.Fatalf("Commons must declare its authoritative PRE2 versionIRI exactly once")
    }
    text := string(governance)
    if strings.Count(text, governanceVersionIRI) != 1 {
        t.Fatalf("Governance must declare its authoritative PRE2 versionIRI exactly once")
    }
    if strings.Count(text, frozenImport) != 1 {
        t.Fatalf("Governance must import the authoritative Commons PRE2 versionIRI exactly once")
    }
    if strings.Contains(text, "owl:imports <https://id.exergism.org/ontology/commons> ;") {
        t.Fatalf("Governance versioned artifact must not import mutable Commons current IRI")
    }
}
