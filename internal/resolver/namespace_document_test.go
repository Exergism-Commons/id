package resolver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAdoptedHashNamespacesHaveResolvableDocuments(t *testing.T) {
	h, err := Load("../..", "../../resolver/registry.json")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("../../catalog/namespaces.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Namespaces []struct {
			ID        string `json:"id"`
			Namespace string `json:"namespace"`
			Status    string `json:"status"`
		} `json:"namespaces"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}

	for _, namespace := range catalog.Namespaces {
		if namespace.Status != "adopted" || namespace.Namespace == "" {
			continue
		}
		base := strings.TrimSuffix(namespace.Namespace, "#")
		path := strings.TrimPrefix(base, "https://id.exergism.org")
		route, ok := h.routes[path]
		if !ok {
			t.Errorf("adopted namespace %s has no resolvable document route: %s", namespace.ID, base)
			continue
		}
		hasHTML := false
		for _, rep := range route.Representations {
			if strings.HasPrefix(rep.MediaType, "text/html") {
				hasHTML = true
				break
			}
		}
		if !hasHTML {
			t.Errorf("adopted namespace %s document must expose HTML", namespace.ID)
		}
	}
}
