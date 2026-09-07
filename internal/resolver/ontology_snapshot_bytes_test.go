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
    SourceRepository string `json:"source_repository"`
    SourceCommit     string `json:"source_commit"`
    Snapshots []struct {
        ID                       string `json:"id"`
        VersionIRI               string `json:"version_iri"`
        ProfileIRI               string `json:"profile_iri"`
        SourcePath               string `json:"source_path"`
        SourceGitBlobSHA         string `json:"source_git_blob_sha"`
        CurrentPublicationPath   string `json:"current_publication_path"`
        VersionedPublicationPath string `json:"versioned_publication_path"`
        PublicationGitBlobSHA    string `json:"publication_git_blob_sha"`
        PublicationTransform     string `json:"publication_transform"`
    } `json:"snapshots"`
}

func TestGovernancePublicationBytesMatchManifestWithoutTransformation(t *testing.T) {
    raw, err := os.ReadFile("../../catalog/ontology-snapshots.json")
    if err != nil {
        t.Fatal(err)
    }
    var catalog ontologySnapshotCatalog
    if err := json.Unmarshal(raw, &catalog); err != nil {
        t.Fatal(err)
    }
    if catalog.SourceRepository != "Exergism-Commons/governance" {
        t.Fatalf("unexpected source repository %q", catalog.SourceRepository)
    }
    if catalog.SourceCommit != "517c0df4c905c79ec2b2824231b40ccaa6a1b510" {
        t.Fatalf("unexpected Governance source commit %q", catalog.SourceCommit)
    }
    if len(catalog.Snapshots) != 3 {
        t.Fatalf("expected Commons, Governance and downstream-authority snapshots, got %d", len(catalog.Snapshots))
    }

    for _, snapshot := range catalog.Snapshots {
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
        if snapshot.CurrentPublicationPath != "" {
            current, err := os.ReadFile("../../" + snapshot.CurrentPublicationPath)
            if err != nil {
                t.Fatalf("read %s: %v", snapshot.CurrentPublicationPath, err)
            }
            if got := gitBlobSHA(current); got != snapshot.SourceGitBlobSHA {
                t.Fatalf("current publication %s is not the adopted source blob: got %s want %s", snapshot.ID, got, snapshot.SourceGitBlobSHA)
            }
        }
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
