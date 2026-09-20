package marketplace

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

// HTTPHandler dispatches incoming HTTP requests to the marketplace service.
type HTTPHandler struct {
	svc *Service
	mux *http.ServeMux
}

// NewHTTPHandler constructs an http.Handler serving the marketplace REST API.
func NewHTTPHandler(svc *Service) http.Handler {
	h := &HTTPHandler{
		svc: svc,
		mux: http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *HTTPHandler) registerRoutes() {
	h.mux.HandleFunc("/api/v1/search", h.handleSearch)
	h.mux.HandleFunc("/api/v1/featured", h.handleFeatured)
	h.mux.HandleFunc("/api/v1/trending", h.handleTrending)
	h.mux.HandleFunc("/api/v1/categories", h.handleCategories)
	h.mux.HandleFunc("/api/v1/publishers/", h.handlePublisher)
	h.mux.HandleFunc("/api/v1/updates", h.handleCheckUpdates)
	h.mux.HandleFunc("/api/v1/publish", h.handlePublish)
	h.mux.HandleFunc("/api/v1/plugins/", h.handlePluginDispatch)
}

func (h *HTTPHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	limit := 20
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := 0
	if oStr := q.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := pubmarket.SearchQuery{
		Keyword:   q.Get("q"),
		Category:  pubmarket.Category(q.Get("category")),
		Publisher: q.Get("publisher"),
		Limit:     limit,
		Offset:    offset,
	}

	res, err := h.svc.Search(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) handleFeatured(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 10
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	res, err := h.svc.GetFeatured(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) handleTrending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 10
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	res, err := h.svc.GetTrending(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cats, err := h.svc.GetCategories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (h *HTTPHandler) handlePublisher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pubID := strings.TrimPrefix(r.URL.Path, "/api/v1/publishers/")
	if pubID == "" {
		http.Error(w, "publisher ID required", http.StatusBadRequest)
		return
	}
	pub, err := h.svc.GetPublisher(r.Context(), pubID)
	if err != nil {
		if errors.Is(err, pubmarket.ErrNotFound) {
			http.Error(w, "publisher not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func (h *HTTPHandler) handleCheckUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req []pubmarket.UpdateCheckItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updates, err := h.svc.CheckUpdates(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updates)
}

func (h *HTTPHandler) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 50*1024*1024))
	if err != nil {
		http.Error(w, "failed reading body", http.StatusBadRequest)
		return
	}

	vInfo, valResult, err := h.svc.PublishPackage(r.Context(), body)
	if err != nil {
		if errors.Is(err, pubmarket.ErrUnauthorizedPublisher) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": err.Error(), "validation": valResult})
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "validation": valResult})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "validation": valResult})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"version":    vInfo,
		"validation": valResult,
	})
}

func (h *HTTPHandler) handlePluginDispatch(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/plugins/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "plugin ID required", http.StatusBadRequest)
		return
	}

	pluginID := parts[0]

	// /api/v1/plugins/{id}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p, err := h.svc.GetPlugin(r.Context(), pluginID)
		if err != nil {
			if errors.Is(err, pubmarket.ErrNotFound) {
				http.Error(w, "plugin not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, p)
		return
	}

	// /api/v1/plugins/{id}/versions
	if len(parts) == 2 && parts[1] == "versions" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		vers, err := h.svc.GetVersions(r.Context(), pluginID)
		if err != nil {
			if errors.Is(err, pubmarket.ErrNotFound) {
				http.Error(w, "plugin not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, vers)
		return
	}

	// /api/v1/plugins/{id}/reviews
	if len(parts) == 2 && parts[1] == "reviews" {
		if r.Method == http.MethodPost {
			var rev pubmarket.Review
			if err := json.NewDecoder(r.Body).Decode(&rev); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			rev.PluginID = pluginID
			if err := h.svc.SubmitReview(r.Context(), rev); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method == http.MethodGet {
			revs, err := h.svc.GetReviews(r.Context(), pluginID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, revs)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /api/v1/plugins/{id}/ratings
	if len(parts) == 2 && parts[1] == "ratings" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			AuthorID string `json:"author_id"`
			Stars    int    `json:"stars"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		sum, err := h.svc.SubmitRating(r.Context(), pluginID, payload.AuthorID, payload.Stars)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, sum)
		return
	}

	// /api/v1/plugins/{id}/versions/{version}/download
	if len(parts) == 4 && parts[1] == "versions" && parts[3] == "download" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		vParsed, err := version.ParseSemVer(parts[2])
		if err != nil {
			http.Error(w, "invalid version format", http.StatusBadRequest)
			return
		}
		stream, vInfo, err := h.svc.DownloadArtifact(r.Context(), pluginID, vParsed)
		if err != nil {
			if errors.Is(err, pubmarket.ErrNotFound) || errors.Is(err, pubmarket.ErrVersionNotFound) {
				http.Error(w, "version not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Length", strconv.FormatInt(vInfo.ArtifactSize, 10))
		w.Header().Set("ETag", fmt.Sprintf("%q", vInfo.ArtifactDigest))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-%s.tar.gz\"", pluginID, vParsed.String()))

		_, _ = io.Copy(w, stream)
		return
	}

	// /api/v1/plugins/{id}/versions/{version}/dependencies
	if len(parts) == 4 && parts[1] == "versions" && parts[3] == "dependencies" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		vParsed, err := version.ParseSemVer(parts[2])
		if err != nil {
			http.Error(w, "invalid version format", http.StatusBadRequest)
			return
		}
		plan, err := h.svc.ResolveDependencies(r.Context(), pluginID, vParsed)
		if err != nil {
			if errors.Is(err, pubmarket.ErrNotFound) || errors.Is(err, pubmarket.ErrVersionNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			if errors.Is(err, pubmarket.ErrDependencyMissing) || errors.Is(err, pubmarket.ErrDependencyConflict) || errors.Is(err, pubmarket.ErrCircularDependency) {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, plan)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
