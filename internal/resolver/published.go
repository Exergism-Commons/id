package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

// PublishedHandler wraps the semantic resolver and additionally serves every
// representation public_path declared by the registry plus the repository's
// static frontend assets under /assets/. Public paths are dereferenceable
// representation locations; assets are presentation resources and do not
// become semantic authorities.
type PublishedHandler struct {
	core   *Handler
	root   fs.FS
	public map[string]publishedRepresentation
}

type publishedRepresentation struct {
	route Route
	rep   Representation
}

// LoadPublished loads the canonical resolver plus its explicitly published
// representation locations. Production entry points should use this loader.
func LoadPublished(rootDir, registryPath string) (*PublishedHandler, error) {
	core, err := Load(rootDir, registryPath)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, err
	}
	var registry Registry
	if err := json.Unmarshal(raw, &registry); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}

	h := &PublishedHandler{
		core:   core,
		root:   os.DirFS(rootDir),
		public: make(map[string]publishedRepresentation),
	}

	for _, route := range registry.Routes {
		for _, rep := range route.Representations {
			if rep.PublicPath == "" {
				continue
			}
			if !validHTTPPath(rep.PublicPath, false) {
				return nil, fmt.Errorf("route %q has invalid public representation path %q", route.Path, rep.PublicPath)
			}
			if _, exists := core.routes[rep.PublicPath]; exists {
				return nil, fmt.Errorf("public representation path %q collides with a route", rep.PublicPath)
			}
			if _, exists := core.aliases[rep.PublicPath]; exists {
				return nil, fmt.Errorf("public representation path %q collides with an alias", rep.PublicPath)
			}
			if _, exists := h.public[rep.PublicPath]; exists {
				return nil, fmt.Errorf("duplicate public representation path %q", rep.PublicPath)
			}
			h.public[rep.PublicPath] = publishedRepresentation{route: route, rep: rep}
		}
	}

	return h, nil
}

func (h *PublishedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		h.serveAsset(w, r)
		return
	}

	published, ok := h.public[r.URL.Path]
	if !ok {
		h.core.ServeHTTP(w, r)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, acceptable := negotiate(r.Header.Get("Accept"), []Representation{published.rep}); !acceptable {
		w.Header().Set("Vary", "Accept")
		w.Header().Set("Acceptable", acceptableMediaTypes([]Representation{published.rep}))
		http.Error(w, "no acceptable representation", http.StatusNotAcceptable)
		return
	}

	data, err := fs.ReadFile(h.root, published.rep.File)
	if err != nil {
		http.Error(w, "representation unavailable", http.StatusInternalServerError)
		return
	}

	sum := sha256.Sum256(data)
	etag := `"sha256-` + hex.EncodeToString(sum[:]) + `"`

	w.Header().Set("Content-Type", published.rep.MediaType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Link", linkHeader(published.route, published.rep))
	w.Header().Set("Content-Location", published.rep.PublicPath)
	w.Header().Set("ETag", etag)
	if published.route.CacheControl != "" {
		w.Header().Set("Cache-Control", published.route.CacheControl)
	}

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}

func (h *PublishedHandler) serveAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rel := strings.TrimPrefix(r.URL.Path, "/")
	if rel == "assets/" || !fs.ValidPath(rel) {
		http.NotFound(w, r)
		return
	}

	info, err := fs.Stat(h.root, rel)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}

	data, err := fs.ReadFile(h.root, rel)
	if err != nil {
		http.Error(w, "asset unavailable", http.StatusInternalServerError)
		return
	}

	mediaType := mime.TypeByExtension(path.Ext(rel))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	sum := sha256.Sum256(data)
	etag := `"sha256-` + hex.EncodeToString(sum[:]) + `"`
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
