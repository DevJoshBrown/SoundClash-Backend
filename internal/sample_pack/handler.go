package sample_pack

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/pkg/storage/r2"
)

type Handler struct {
	queries *db.Queries
	r2      *r2.Client
}

func NewHandler(queries *db.Queries, r2 *r2.Client) *Handler {
	return &Handler{queries: queries, r2: r2}
}

func (h Handler) UploadSamplePack(w http.ResponseWriter, r *http.Request) {
	// 100MB limit for zip files
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	genres := r.Form["genres"]
	if len(genres) == 0 {
		http.Error(w, "at least one genre is required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext != ".zip" {
		http.Error(w, "bad type, fileext must be .zip", http.StatusBadRequest)
		return
	}

	key := "sample-packs" + name + ".zip"
	if err := h.r2.Upload(r.Context(), key, file, "application/zip"); err != nil {
		http.Error(w, "failed to upload file", http.StatusInternalServerError)
		return
	}

	fileURL := h.r2.PublicURL(key)

	pack, err := h.queries.CreateSamplePack(r.Context(), db.CreateSamplePackParams{
		Name:    name,
		Genres:  genres,
		FileUrl: fileURL,
	})
	if err != nil {
		http.Error(w, "failed to create sample pack", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pack)
}

func (h Handler) ListSamplePacks(w http.ResponseWriter, r *http.Request) {
	packs, err := h.queries.ListSamplePacks(r.Context())
	if err != nil {
		http.Error(w, "failed to list sample packs", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packs)
}
