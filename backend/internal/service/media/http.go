// Package media implements policy-controlled, streamed uploads and public files.
package media

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/media"
	"peptide-site/internal/conf"
	repo "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/filestaging"
	admin "peptide-site/internal/service/admin"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	Repo   *repo.Repo
	Config conf.Config
	Root   string
	Dir    string
}

func NewHandler(r *repo.Repo, c conf.Config, root string) (*Handler, error) {
	dir, err := filepath.EvalSymlinks(c.Data.Storage.UploadDir)
	if err != nil {
		return nil, err
	}
	return &Handler{r, c, root, dir}, nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/media")
	objectID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	switch {
	case path == "/upload-policy" && r.Method == "GET":
		version := uint64(0)
		if s, ok := configreload.FromContext(r.Context()); ok {
			version = s.Status.Version
		}
		admin.Reply(w, 200, map[string]any{"version": version, "allowedTypes": h.Config.Data.Storage.AllowedTypes, "maxImageBytes": h.Config.Data.Storage.MaxImageBytes, "maxVideoBytes": h.Config.Data.Storage.MaxVideoBytes, "maxDocumentBytes": h.Config.Data.Storage.MaxDocumentBytes})
	case path == "" && r.Method == "GET":
		page, size := 1, 20
		var err error
		if p := r.URL.Query().Get("page"); p != "" {
			page, err = strconv.Atoi(p)
		}
		if err == nil {
			if p := r.URL.Query().Get("pageSize"); p != "" {
				size, err = strconv.Atoi(p)
			}
		}
		if err != nil || page < 1 || page > 10000000 || size < 1 || size > 100 || len(r.URL.Query().Get("q")) > 200 {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		rows, total, err := h.Repo.List(r.Context(), page, size, r.URL.Query().Get("type"), r.URL.Query().Get("q"))
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]any{"items": rows, "total": total, "page": page, "pageSize": size})
	case path == "" && r.Method == "POST":
		h.upload(w, r)
	case objectID > 0 && strings.HasSuffix(path, "/references") && r.Method == "GET":
		rows, err := h.Repo.References(r.Context(), objectID)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]any{"items": rows})
	case objectID > 0 && r.Method == "GET":
		row, err := h.Repo.Get(r.Context(), objectID)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, row)
	case objectID > 0 && r.Method == "PATCH":
		var in struct {
			Name string `json:"name"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&in) != nil || decoder.Decode(new(any)) != io.EOF {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		row, err := h.Repo.Rename(r.Context(), objectID, in.Name)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, row)
	case objectID > 0 && r.Method == "DELETE":
		if err := h.Repo.Delete(r.Context(), objectID); err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]bool{"ok": true})
	default:
		admin.Failure(w, r, errdef.Invalid())
	}
}
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	policy := h.Config.Data.Storage
	maxSize := max(policy.MaxImageBytes, policy.MaxVideoBytes, policy.MaxDocumentBytes)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize+(1<<20))
	multipart, err := r.MultipartReader()
	if err != nil {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	part, err := multipart.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	prefix, err := io.ReadAll(io.LimitReader(part, 512))
	if err != nil || len(prefix) == 0 {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	mime := http.DetectContentType(prefix)
	allowed := false
	for _, value := range policy.AllowedTypes {
		if value == mime {
			allowed = true
		}
	}
	if !allowed {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif", "video/mp4": ".mp4", "video/webm": ".webm", "application/pdf": ".pdf"}
	kind := "document"
	limit := policy.MaxDocumentBytes
	if strings.HasPrefix(mime, "image/") {
		kind = "image"
		limit = policy.MaxImageBytes
	} else if strings.HasPrefix(mime, "video/") {
		kind = "video"
		limit = policy.MaxVideoBytes
	}
	temporary, err := os.CreateTemp(h.Dir, ".upload-")
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	temporaryPath := temporary.Name()
	releaseTemporary := filestaging.Protect(temporaryPath)
	defer releaseTemporary()
	defer func() { _ = temporary.Close(); _ = os.Remove(temporaryPath) }()
	size, err := io.Copy(temporary, io.LimitReader(io.MultiReader(bytes.NewReader(prefix), part), limit+1))
	if err != nil || size <= 0 || size > limit {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if _, err := multipart.NextPart(); err != io.EOF {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	var width, height *int
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		admin.Failure(w, r, err)
		return
	}
	if kind == "image" {
		dimensions, _, err := image.DecodeConfig(temporary)
		if err != nil || dimensions.Width <= 0 || dimensions.Height <= 0 || int64(dimensions.Width)*int64(dimensions.Height) > 20000000 {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		_, _ = temporary.Seek(0, io.SeekStart)
		if _, _, err := image.Decode(temporary); err != nil {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		width, height = &dimensions.Width, &dimensions.Height
	}
	if mime == "video/mp4" && !validMP4(temporary, size) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if mime == "video/webm" && !bytes.Contains(prefix, []byte("webm")) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if err := temporary.Sync(); err != nil {
		admin.Failure(w, r, err)
		return
	}
	if err := temporary.Close(); err != nil {
		admin.Failure(w, r, err)
		return
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		admin.Failure(w, r, err)
		return
	}
	key := hex.EncodeToString(raw) + extensions[mime]
	destination := filepath.Join(h.Dir, key)
	releaseDestination := filestaging.Protect(destination)
	defer releaseDestination()
	if err := os.Rename(temporaryPath, destination); err != nil {
		admin.Failure(w, r, err)
		return
	}
	saved := false
	defer func() {
		if !saved {
			_ = os.Remove(destination)
		}
	}()
	name := filepath.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
	if len(name) > 255 {
		name = key
	}
	now := time.Now().UTC()
	i := auth.Current(r.Context())
	var uploader *int64
	if i != nil {
		uploader = &i.User.ID
	}
	row := biz.Media{Name: name, URL: "/uploads/" + h.Root + "/" + key, MIME: mime, MediaType: kind, Size: size, Width: width, Height: height, Available: true, StorageRootID: h.Root, StorageKey: key, UploadedBy: uploader, CreatedAt: now, UpdatedAt: now}
	if err := h.Repo.Save(r.Context(), &row); err != nil {
		admin.Failure(w, r, err)
		return
	}
	saved = true
	admin.Reply(w, 201, row)
}
func validMP4(file *os.File, size int64) bool {
	var offset int64
	media, metadata := false, false
	for boxes := 0; offset < size && boxes < 4096; boxes++ {
		var header [16]byte
		if _, err := file.ReadAt(header[:8], offset); err != nil {
			return false
		}
		length := int64(binary.BigEndian.Uint32(header[:4]))
		headerSize := int64(8)
		if length == 1 {
			if _, err := file.ReadAt(header[8:], offset+8); err != nil {
				return false
			}
			extended := binary.BigEndian.Uint64(header[8:])
			if extended > uint64(size) {
				return false
			}
			length = int64(extended)
			headerSize = 16
		} else if length == 0 {
			length = size - offset
		}
		if length < headerSize || length > size-offset {
			return false
		}
		switch string(header[4:8]) {
		case "mdat":
			media = true
		case "moov":
			metadata = true
		}
		offset += length
	}
	return offset == size && media && metadata
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
