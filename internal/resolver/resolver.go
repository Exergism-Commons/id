package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Registry struct {
	Authority string  `json:"authority"`
	Routes    []Route `json:"routes"`
}

type Route struct {
	Path            string           `json:"path"`
	Aliases         []string         `json:"aliases,omitempty"`
	Canonical       string           `json:"canonical"`
	CacheControl    string           `json:"cache_control,omitempty"`
	Representations []Representation `json:"representations"`
}

type Representation struct {
	MediaType  string `json:"media_type"`
	File       string `json:"file"`
	PublicPath string `json:"public_path,omitempty"`
	Default    bool   `json:"default,omitempty"`
}

type Handler struct {
	root    fs.FS
	routes  map[string]Route
	aliases map[string]string
}

func Load(rootDir, registryPath string) (*Handler, error) {
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, err
	}

	var registry Registry
	if err := json.Unmarshal(raw, &registry); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if registry.Authority == "" {
		return nil, errors.New("registry authority is required")
	}

	h := &Handler{
		root:    os.DirFS(rootDir),
		routes:  make(map[string]Route),
		aliases: make(map[string]string),
	}

	for _, route := range registry.Routes {
		if err := validateRoute(route); err != nil {
			return nil, err
		}
		for _, rep := range route.Representations {
			info, err := fs.Stat(h.root, rep.File)
			if err != nil {
				return nil, fmt.Errorf("route %q representation %q: %w", route.Path, rep.File, err)
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("route %q representation %q is not a regular file", route.Path, rep.File)
			}
		}
		if _, exists := h.routes[route.Path]; exists {
			return nil, fmt.Errorf("duplicate route %q", route.Path)
		}
		h.routes[route.Path] = route
		for _, alias := range route.Aliases {
			if _, exists := h.routes[alias]; exists {
				return nil, fmt.Errorf("alias %q collides with route", alias)
			}
			if _, exists := h.aliases[alias]; exists {
				return nil, fmt.Errorf("duplicate alias %q", alias)
			}
			h.aliases[alias] = route.Path
		}
	}

	return h, nil
}

func validateRoute(route Route) error {
	if !validHTTPPath(route.Path, false) {
		return fmt.Errorf("invalid route path %q", route.Path)
	}
	if route.Canonical == "" {
		return fmt.Errorf("route %q has no canonical IRI", route.Path)
	}
	if len(route.Representations) == 0 {
		return fmt.Errorf("route %q has no representations", route.Path)
	}

	defaults := 0
	for _, rep := range route.Representations {
		mediaType, _, err := mime.ParseMediaType(rep.MediaType)
		if err != nil || !strings.Contains(mediaType, "/") {
			return fmt.Errorf("route %q has invalid media type %q", route.Path, rep.MediaType)
		}
		if rep.File == "" || !fs.ValidPath(rep.File) {
			return fmt.Errorf("route %q has invalid file %q", route.Path, rep.File)
		}
		if rep.Default {
			defaults++
		}
	}
	if defaults > 1 {
		return fmt.Errorf("route %q has multiple default representations", route.Path)
	}

	for _, alias := range route.Aliases {
		if !validHTTPPath(alias, true) {
			return fmt.Errorf("route %q has invalid alias %q", route.Path, alias)
		}
	}

	return nil
}

func validHTTPPath(value string, allowTrailingSlash bool) bool {
	if value == "" || !strings.HasPrefix(value, "/") {
		return false
	}
	if value == "/" {
		return true
	}
	if strings.HasSuffix(value, "/") {
		if !allowTrailingSlash {
			return false
		}
		trimmed := strings.TrimSuffix(value, "/")
		return trimmed != "" && path.Clean(trimmed) == trimmed
	}
	return path.Clean(value) == value
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if canonicalPath, ok := h.aliases[r.URL.Path]; ok {
		target := canonicalPath
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
		return
	}

	route, ok := h.routes[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	rep, ok := negotiate(r.Header.Get("Accept"), route.Representations)
	if !ok {
		w.Header().Set("Vary", "Accept")
		w.Header().Set("Acceptable", acceptableMediaTypes(route.Representations))
		http.Error(w, "no acceptable representation", http.StatusNotAcceptable)
		return
	}

	data, err := fs.ReadFile(h.root, rep.File)
	if err != nil {
		http.Error(w, "representation unavailable", http.StatusInternalServerError)
		return
	}

	sum := sha256.Sum256(data)
	etag := `"sha256-` + hex.EncodeToString(sum[:]) + `"`

	w.Header().Set("Content-Type", rep.MediaType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Link", linkHeader(route, rep))
	w.Header().Set("ETag", etag)
	if route.CacheControl != "" {
		w.Header().Set("Cache-Control", route.CacheControl)
	}
	if len(route.Representations) > 1 {
		w.Header().Set("Vary", "Accept")
	}
	if rep.PublicPath != "" {
		w.Header().Set("Content-Location", rep.PublicPath)
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

func linkHeader(route Route, selected Representation) string {
	links := []string{fmt.Sprintf("<%s>; rel=\"canonical\"", route.Canonical)}
	for _, rep := range route.Representations {
		if rep.PublicPath == "" || rep.PublicPath == selected.PublicPath {
			continue
		}
		baseType, _, _ := mime.ParseMediaType(rep.MediaType)
		links = append(links, fmt.Sprintf("<%s>; rel=\"alternate\"; type=\"%s\"", rep.PublicPath, baseType))
	}
	return strings.Join(links, ", ")
}

func acceptableMediaTypes(reps []Representation) string {
	values := make([]string, 0, len(reps))
	for _, rep := range reps {
		mediaType, _, _ := mime.ParseMediaType(rep.MediaType)
		values = append(values, mediaType)
	}
	sort.Strings(values)
	return strings.Join(values, ", ")
}

type acceptRange struct {
	typ         string
	subtype     string
	q           float64
	specificity int
	order       int
}

func negotiate(header string, reps []Representation) (Representation, bool) {
	if strings.TrimSpace(header) == "" {
		return defaultRepresentation(reps), true
	}

	ranges := parseAccept(header)
	bestRep := -1
	bestQ := -1.0
	bestSpecificity := -1
	bestOrder := int(^uint(0) >> 1)

	for i, rep := range reps {
		mediaType, _, err := mime.ParseMediaType(rep.MediaType)
		if err != nil {
			continue
		}
		parts := strings.SplitN(strings.ToLower(mediaType), "/", 2)
		if len(parts) != 2 {
			continue
		}

		repQ := -1.0
		repSpecificity := -1
		repOrder := int(^uint(0) >> 1)
		for _, ar := range ranges {
			if !mediaMatches(ar, parts[0], parts[1]) {
				continue
			}
			if ar.specificity > repSpecificity || (ar.specificity == repSpecificity && ar.order < repOrder) {
				repQ = ar.q
				repSpecificity = ar.specificity
				repOrder = ar.order
			}
		}
		if repQ <= 0 {
			continue
		}

		if repQ > bestQ ||
			(repQ == bestQ && repSpecificity > bestSpecificity) ||
			(repQ == bestQ && repSpecificity == bestSpecificity && repOrder < bestOrder) ||
			(repQ == bestQ && repSpecificity == bestSpecificity && repOrder == bestOrder && bestRep == -1) {
			bestRep = i
			bestQ = repQ
			bestSpecificity = repSpecificity
			bestOrder = repOrder
		}
	}

	if bestRep < 0 {
		return Representation{}, false
	}
	return reps[bestRep], true
}

func defaultRepresentation(reps []Representation) Representation {
	for _, rep := range reps {
		if rep.Default {
			return rep
		}
	}
	return reps[0]
}

func parseAccept(header string) []acceptRange {
	parts := strings.Split(header, ",")
	result := make([]acceptRange, 0, len(parts))
	for order, raw := range parts {
		mediaRange, params, err := mime.ParseMediaType(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		split := strings.SplitN(strings.ToLower(mediaRange), "/", 2)
		if len(split) != 2 {
			continue
		}
		q := 1.0
		if rawQ, ok := params["q"]; ok {
			parsed, err := strconv.ParseFloat(rawQ, 64)
			if err != nil || parsed < 0 || parsed > 1 {
				continue
			}
			q = parsed
		}
		specificity := 2
		if split[0] == "*" && split[1] == "*" {
			specificity = 0
		} else if split[1] == "*" {
			specificity = 1
		}
		result = append(result, acceptRange{
			typ: split[0], subtype: split[1], q: q, specificity: specificity, order: order,
		})
	}
	return result
}

func mediaMatches(ar acceptRange, typ, subtype string) bool {
	if ar.typ == "*" && ar.subtype == "*" {
		return true
	}
	if ar.typ != typ {
		return false
	}
	return ar.subtype == "*" || ar.subtype == subtype
}
