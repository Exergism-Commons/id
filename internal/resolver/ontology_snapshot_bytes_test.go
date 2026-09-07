package resolver

import (
    "crypto/sha1"
    "fmt"
    "os"
    "testing"
)

func gitBlobSHA(data []byte) string {
    header := []byte(fmt.Sprintf("blob %d\x00", len(data)))
    sum := sha1.Sum(append(header, data...))
    return fmt.Sprintf("%x", sum)
}

func TestImmutableGovernanceOntologyBytesMatchPinnedUpstreamBlobs(t *testing.T) {
    expected := map[string]string{
        "../../representations/commons.ttl": "09d2965185a106e6c1428a6e0dba40dd81b640d2",
        "../../representations/governance.ttl": "35cb8e846aa60688ac90710fb4d00e35e1590850",
    }
    for path, want := range expected {
        data, err := os.ReadFile(path)
        if err != nil {
            t.Fatal(err)
        }
        if got := gitBlobSHA(data); got != want {
            t.Fatalf("immutable ontology representation %s drifted: got %s want %s", path, got, want)
        }
    }
}
