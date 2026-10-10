// Package configreload publishes complete, leased runtime generations.
package configreload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"

	"peptide-site/internal/conf"
)

// Resource is prepared independently, then committed before publication.
// Close must release every candidate resource, including after a failed commit.
type Resource struct {
	Handler  http.Handler
	Identity string
	Schema   string
	Commit   func() error
	Close    func()
}
type Factory func(context.Context, conf.Config) (*Resource, error)

type generation struct {
	config   conf.Config
	resource *Resource
	leases   sync.WaitGroup
	version  uint64
}

// Status contains safe operational metadata; no configuration values.
type Status struct {
	Version   uint64            `json:"version"`
	LoadedAt  time.Time         `json:"loadedAt"`
	CheckedAt time.Time         `json:"checkedAt"`
	Error     string            `json:"error,omitempty"`
	Sources   map[string]string `json:"sources"`
}
type contextKey struct{}

// Snapshot is fixed for one request, including its configuration status.
type Snapshot struct {
	Config conf.Config
	Status Status
}

func FromContext(ctx context.Context) (Snapshot, bool) {
	s, ok := ctx.Value(contextKey{}).(Snapshot)
	return s, ok
}

// Manager serializes resource changes and pins requests to a generation.
type Manager struct {
	mu       sync.Mutex
	reloadMu sync.Mutex
	current  *generation
	status   Status
	path     string
	factory  Factory
	loader   *conf.Loader
	hash     string
	closed   bool
	draining sync.WaitGroup
	retired  map[*generation]struct{}
}

func New(ctx context.Context, path string, c conf.Config, loader *conf.Loader, f Factory) (*Manager, error) {
	r, err := f(ctx, c)
	if err != nil {
		return nil, err
	}
	if r.Commit != nil {
		if err = r.Commit(); err != nil {
			r.Close()
			return nil, err
		}
	}
	now := time.Now().UTC()
	m := &Manager{retired: map[*generation]struct{}{}, current: &generation{config: c, resource: r, version: 1}, path: path, factory: f, loader: loader, status: Status{Version: 1, LoadedAt: now, CheckedAt: now, Sources: loader.Sources()}}

	return m, nil
}
func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.status
	s.Sources = cloneSources(s.Sources)
	return s
}
func (m *Manager) Config() conf.Config { m.mu.Lock(); defer m.mu.Unlock(); return m.current.config }
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		http.Error(w, "service unavailable", 503)
		return
	}
	g := m.current
	g.leases.Add(1)
	s := m.status
	s.Sources = cloneSources(s.Sources)
	m.mu.Unlock()
	defer g.leases.Done()
	ctx, cancel := context.WithTimeout(r.Context(), g.config.Server.HTTP.Timeout)
	defer cancel()
	ctx = context.WithValue(ctx, contextKey{}, Snapshot{g.config, s})
	// Deadlines apply to this request rather than a stale net/http.Server copy.
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(min(g.config.Server.HTTP.ReadTimeout, g.config.Server.HTTP.Timeout)))
	_ = controller.SetWriteDeadline(time.Now().Add(min(g.config.Server.HTTP.WriteTimeout, g.config.Server.HTTP.Timeout)))
	g.resource.Handler.ServeHTTP(w, r.WithContext(ctx))
}

// Reload checks contents, prepares a complete candidate and publishes atomically.
// prepareListener prebinds a changed address; activation must not fail.
func (m *Manager) Reload(ctx context.Context, prepareListener func(conf.Config) (activate func(), rollback func(), err error)) error {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	body, err := os.ReadFile(m.path)
	if err != nil {
		return m.failed("configuration file unavailable")
	}
	digest := hash(body)
	m.mu.Lock()
	same := digest == m.hash
	old := m.current
	closed := m.closed
	m.status.CheckedAt = time.Now().UTC()
	m.mu.Unlock()
	if closed {
		return errors.New("configuration manager closed")
	}
	if same {
		m.mu.Lock()
		m.status.Error = ""
		m.mu.Unlock()
		return nil
	}
	c, err := m.loader.Decode(body)
	if err != nil {
		return m.failed(err.Error())
	}
	if reflect.DeepEqual(c, old.config) {
		m.mu.Lock()
		m.hash = digest
		m.status.Sources = m.loader.Sources()
		m.status.Error = ""
		m.mu.Unlock()
		return nil
	}
	// In-flight old generations may still encrypt secrets. Keep their active
	// keys available until every lease has drained; stored ciphertext is checked
	// separately by the candidate auth repository.
	m.mu.Lock()
	preserve := func(g *generation) bool {
		id := g.config.Auth.TOTP.ActiveKeyID
		return id == "" || c.Auth.TOTP.Keys[id] == g.config.Auth.TOTP.Keys[id]
	}
	keysSafe := preserve(old)
	for g := range m.retired {
		if !preserve(g) {
			keysSafe = false
		}
	}
	m.mu.Unlock()
	if !keysSafe {
		return m.failed("active key is still required by a runtime generation")
	}
	r, err := m.factory(ctx, c)
	if err != nil {
		return m.failed("resource preparation failed")
	}
	committed := false
	defer func() {
		if !committed {
			r.Close()
		}
	}()
	if r.Identity != old.resource.Identity || r.Schema != old.resource.Schema {
		return m.failed("database site identity or schema mismatch")
	}
	activate, rollback, err := prepareListener(c)
	if err != nil {
		return m.failed("HTTP address unavailable")
	}
	defer func() {
		if !committed && rollback != nil {
			rollback()
		}
	}()
	if r.Commit != nil {
		if err = r.Commit(); err != nil {
			return m.failed("resource commit failed")
		}
	}
	m.mu.Lock()
	m.retired[old] = struct{}{}
	m.current = &generation{config: c, resource: r, version: old.version + 1}
	m.hash = digest
	m.status.Version = m.current.version
	m.status.LoadedAt = time.Now().UTC()
	m.status.Error = ""
	m.status.Sources = m.loader.Sources()
	if activate != nil {
		activate()
	}
	m.mu.Unlock()
	committed = true
	m.draining.Add(1)
	go func() {
		defer m.draining.Done()
		old.leases.Wait()
		old.resource.Close()
		m.mu.Lock()
		delete(m.retired, old)
		m.mu.Unlock()
	}()
	slog.Info("configuration applied", "version", old.version+1, "app", c.App.Name)
	return nil
}
func (m *Manager) failed(message string) error {
	m.mu.Lock()
	m.status.Error = message
	m.status.CheckedAt = time.Now().UTC()
	m.mu.Unlock()
	slog.Warn("configuration rejected", "reason", message)
	return errors.New(message)
}
func (m *Manager) Watch(ctx context.Context, prepare func(conf.Config) (func(), func(), error)) {
	for {
		timer := time.NewTimer(m.Config().Reload.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			_ = m.Reload(ctx, prepare)
		}
	}
}
func (m *Manager) Close() {
	m.reloadMu.Lock()
	m.mu.Lock()
	m.closed = true
	g := m.current
	m.mu.Unlock()
	m.reloadMu.Unlock()
	g.leases.Wait()
	g.resource.Close()
	m.draining.Wait()
}

func cloneSources(source map[string]string) map[string]string {
	r := make(map[string]string, len(source))
	for k, v := range source {
		r[k] = v
	}
	return r
}
