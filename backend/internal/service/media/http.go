// Package media serves image library, directories, COS uploads and retained local files.
package media

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"io"
	"net/http"
	"path/filepath"
	"peptide-site/internal/conf"
	repo "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/pkg/cosstore"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/imageformat"
	admin "peptide-site/internal/service/admin"
	"strconv"
	"strings"
)

type Handler struct {
	Repo   *repo.Repo
	Config conf.Config
}

func NewHandler(r *repo.Repo, c conf.Config) (*Handler, error) {
	if c.Data.Storage.COS.SecretID != "" && c.Data.Storage.COS.SecretKey != "" {
		r.Store = cosstore.New(c.Data.Storage.COS)
	}
	return &Handler{Repo: r, Config: c}, nil
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32768)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		return errdef.Invalid()
	}
	return nil
}
func allowedExtensions(configured []string) map[string]string {
	out := map[string]string{}
	for _, value := range configured {
		for _, ext := range imageformat.Extensions(value) {
			if _, exists := out[ext]; !exists {
				out[ext] = value
			}
		}
	}
	return out
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/media")
	ctx := r.Context()
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var result any = map[string]bool{"ok": true}
	var err error
	status := 200
	switch {
	case path == "/upload-policy" && r.Method == "GET":
		version := uint64(0)
		if s, ok := configreload.FromContext(ctx); ok {
			version = s.Status.Version
		}
		result = map[string]any{"version": version, "enabled": h.Config.Data.Storage.COS.Enabled, "allowedTypes": h.Config.Data.Storage.AllowedTypes, "allowedExtensions": allowedExtensions(h.Config.Data.Storage.AllowedTypes), "maxImageBytes": h.Config.Data.Storage.MaxImageBytes, "maxBatchFiles": 20}
	case path == "/upload/init" && r.Method == "POST":
		var in repo.UploadInput
		if err = decode(w, r, &in); err == nil {
			result, err = h.Repo.InitUpload(ctx, h.Config.Data.Storage, in)
			status = 201
		}
	case path == "/upload/refresh" && r.Method == "POST":
		var in struct {
			BatchID string `json:"batchId"`
		}
		if err = decode(w, r, &in); err == nil {
			result, err = h.Repo.RefreshUpload(ctx, in.BatchID)
		}
	case id > 0 && path == "/"+strconv.FormatInt(id, 10)+"/complete" && r.Method == "POST":
		result, err = h.Repo.CompleteUpload(ctx, id)
	case id > 0 && path == "/"+strconv.FormatInt(id, 10)+"/cancel" && r.Method == "POST":
		err = h.Repo.CancelUpload(ctx, id)
	case path == "/groups" && r.Method == "GET":
		var rows []repo.Group
		rows, err = h.Repo.Groups(ctx)
		result = map[string]any{"items": rows}
	case (path == "/groups" && r.Method == "POST") || (strings.HasPrefix(path, "/groups/") && id > 0 && r.Method == "PATCH"):
		var in struct {
			Name string `json:"name"`
		}
		if err = decode(w, r, &in); err == nil {
			result, err = h.Repo.SaveGroup(ctx, id, in.Name)
		}
	case strings.HasPrefix(path, "/groups/") && id > 0 && r.Method == "DELETE":
		err = h.Repo.DeleteGroup(ctx, id)
	case (path == "/batch-move" || path == "/batch-delete") && r.Method == "POST":
		var in struct {
			IDs     []string `json:"ids"`
			GroupID *int64   `json:"groupId,string"`
		}
		if err = decode(w, r, &in); err == nil {
			ids := make([]int64, 0, len(in.IDs))
			for _, raw := range in.IDs {
				var n int64
				n, err = strconv.ParseInt(raw, 10, 64)
				if err != nil || n <= 0 {
					err = errdef.Invalid()
					break
				}
				ids = append(ids, n)
			}
			if err == nil {
				if path == "/batch-move" {
					err = h.Repo.Move(ctx, ids, in.GroupID)
				} else {
					err = h.Repo.DeleteBatch(ctx, ids)
				}
			}
		}
	case path == "" && r.Method == "GET":
		page, size := 1, 24
		if p := r.URL.Query().Get("page"); p != "" {
			page, err = strconv.Atoi(p)
		}
		if p := r.URL.Query().Get("pageSize"); p != "" && err == nil {
			size, err = strconv.Atoi(p)
		}
		group := r.URL.Query().Get("groupId")
		if group != "" && group != "ungrouped" {
			n, e := strconv.ParseInt(group, 10, 64)
			if e != nil || n <= 0 {
				err = errdef.Invalid()
			}
		}
		if err != nil || page < 1 || page > 10000000 || size < 1 || size > 100 || len(r.URL.Query().Get("q")) > 200 {
			err = errdef.Invalid()
			break
		}
		rows, total, e := h.Repo.List(ctx, page, size, r.URL.Query().Get("type"), r.URL.Query().Get("q"), group)
		err = e
		result = map[string]any{"items": rows, "total": total, "page": page, "pageSize": size}
	case id > 0 && strings.HasSuffix(path, "/references") && r.Method == "GET":
		rows, e := h.Repo.References(ctx, id)
		err = e
		result = map[string]any{"items": rows}
	case id > 0 && r.Method == "GET":
		result, err = h.Repo.Get(ctx, id)
	case id > 0 && r.Method == "PATCH":
		var in struct {
			Name string `json:"name"`
		}
		if err = decode(w, r, &in); err == nil {
			result, err = h.Repo.Rename(ctx, id, in.Name)
		}
	case id > 0 && r.Method == "DELETE":
		err = h.Repo.Delete(ctx, id)
	default:
		err = errdef.Invalid()
	}
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	admin.Reply(w, status, result)
}
func (h *Handler) PublicFiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/uploads/"), "/")
		if len(parts) != 2 || len(parts[0]) != 32 || parts[1] == "" || filepath.Base(parts[1]) != parts[1] || strings.Contains(parts[1], "\\") || strings.HasPrefix(parts[1], ".") {
			http.NotFound(w, r)
			return
		}
		row, path, err := h.Repo.FindFile(r.Context(), parts[0], parts[1])
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		w.Header().Set("Content-Type", row.MIME)
		http.ServeFile(w, r, path)
	})
}
