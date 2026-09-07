package resolver

import (
    "crypto/sha1"
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

func TestImmutableGovernanceOntologySnapshotsArePinnedAndClosureFrozen(t *testing.T) {
    commonsPath := "../../representations/snapshots/commons-06e614c21f9623658c16175a381279f9c36ef526.ttl"
    commons, err := os.ReadFile(commonsPath)
    if err != nil {
        t.Fatal(err)
    }
    if got, want := gitBlobSHA(commons), "09d2965185a106e6c1428a6e0dba40dd81b640d2"; got != want {
        t.Fatalf("immutable Commons snapshot drifted: got %s want %s", got, want)
    }

    governancePath := "../../representations/snapshots/governance-06e614c21f9623658c16175a381279f9c36ef526.ttl"
    governance, err := os.ReadFile(governancePath)
    if err != nil {
        t.Fatal(err)
    }
    if got, want := gitBlobSHA(governance), "9b9b9ea49962166b56d31c5b8893020641ad4b94"; got != want {
        t.Fatalf("immutable Governance publication snapshot drifted: got %s want %s", got, want)
    }

    frozenImport := "owl:imports <https://id.exergism.org/ontology/commons/06e614c21f9623658c16175a381279f9c36ef526> ;"
    sourceImport := "owl:imports <https://id.exergism.org/ontology/commons> ;"
    text := string(governance)
    if strings.Count(text, frozenImport) != 1 {
        t.Fatalf("Governance snapshot must contain exactly one closure-frozen Commons import")
    }
    if strings.Contains(text, sourceImport) {
        t.Fatalf("Governance snapshot must not retain the mutable Commons import")
    }

    reconstructedSource := []byte(strings.Replace(text, frozenImport, sourceImport, 1))
    if got, want := gitBlobSHA(reconstructedSource), "35cb8e846aa60688ac90710fb4d00e35e1590850"; got != want {
        t.Fatalf("Governance snapshot is not the pinned upstream blob plus only the documented import rewrite: got reconstructed source %s want %s", got, want)
    }
}
