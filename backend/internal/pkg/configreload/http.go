package configreload

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"peptide-site/internal/conf"
)

// HTTP owns listener generations; each connection obtains current header/idle
// deadlines, while the handler pins the body and response to one generation.
type HTTP struct {
	manager *Manager
	mu      sync.Mutex
	servers map[*http.Server]struct{}
	address string
	cancel  context.CancelFunc
	watcher sync.WaitGroup
	drains  sync.WaitGroup
}

func NewHTTP(m *Manager) *HTTP { return &HTTP{manager: m, servers: map[*http.Server]struct{}{}} }
func (h *HTTP) Start(ctx context.Context) error {
	activate, rollback, err := h.prepare(h.manager.Config())
	if err != nil {
		return err
	}
	if activate != nil {
		activate()
	} else if rollback != nil {
		rollback()
	}
	watchCtx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.watcher.Add(1)
	go func() { defer h.watcher.Done(); h.manager.Watch(watchCtx, h.prepare) }()
	return nil
}
func (h *HTTP) prepare(c conf.Config) (func(), func(), error) {
	h.mu.Lock()
	same := h.address == c.Server.HTTP.Addr
	h.mu.Unlock()
	if same {
		return nil, nil, nil
	}
	listener, err := net.Listen("tcp", c.Server.HTTP.Addr)
	if err != nil {
		return nil, nil, err
	}
	srv := &http.Server{Handler: h.manager, ConnState: func(conn net.Conn, state http.ConnState) {
		dc, ok := conn.(*deadlineConn)
		if !ok {
			return
		}
		switch state {
		case http.StateIdle:
			dc.phase.Store(2)
			_ = dc.SetReadDeadline(time.Now().Add(h.manager.Config().Server.HTTP.IdleTimeout))
		case http.StateActive:
			dc.phase.Store(0)
			cfg := h.manager.Config().Server.HTTP
			_ = dc.SetReadDeadline(time.Now().Add(cfg.ReadTimeout))
			_ = dc.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
		}
	}}
	// Activation only changes ownership and starts a prebound listener. It cannot
	// block on Manager.Config because it runs under the publication lock.
	activate := func() {
		h.mu.Lock()
		old := make([]*http.Server, 0, len(h.servers))
		for s := range h.servers {
			old = append(old, s)
		}
		h.servers[srv] = struct{}{}
		h.address = c.Server.HTTP.Addr
		h.mu.Unlock()
		go func() { _ = srv.Serve(&deadlineListener{Listener: listener, manager: h.manager}) }()
		for _, s := range old {
			h.drains.Add(1)
			go func(s *http.Server) {
				defer h.drains.Done()
				_ = s.Shutdown(context.Background())
				h.mu.Lock()
				delete(h.servers, s)
				h.mu.Unlock()
			}(s)
		}
	}
	return activate, func() { _ = listener.Close() }, nil
}
func (h *HTTP) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	h.watcher.Wait()
	h.mu.Lock()
	servers := make([]*http.Server, 0, len(h.servers))
	for s := range h.servers {
		servers = append(servers, s)
	}
	h.mu.Unlock()
	for _, s := range servers {
		if err := s.Shutdown(ctx); err != nil {
			_ = s.Close()
		}
	}
	h.drains.Wait()
	h.manager.Close()
	return nil
}

type deadlineListener struct {
	net.Listener
	manager *Manager
}

func (l *deadlineListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	dc := &deadlineConn{Conn: c, manager: l.manager}
	dc.phase.Store(1)
	return dc, nil
}

type deadlineConn struct {
	net.Conn
	manager *Manager
	phase   atomic.Int32
}

func (c *deadlineConn) Read(p []byte) (int, error) {
	phase := c.phase.Swap(0)
	if phase == 1 {
		cfg := c.manager.Config().Server.HTTP
		_ = c.SetReadDeadline(time.Now().Add(cfg.ReadHeaderTimeout))
		_ = c.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
	}
	n, err := c.Conn.Read(p)
	if phase == 2 && n > 0 {
		c.phase.Store(1)
		_ = c.SetReadDeadline(time.Now().Add(c.manager.Config().Server.HTTP.ReadHeaderTimeout))
	}
	return n, err
}

// Reload triggers the same transaction as the periodic watcher.
func (h *HTTP) Reload(ctx context.Context) error { return h.manager.Reload(ctx, h.prepare) }

// net/http clears deadlines while transitioning from keepalive to headers.
// Preserve the configured phase deadline until the request becomes active.
func (c *deadlineConn) SetReadDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		switch c.phase.Load() {
		case 1:
			deadline = time.Now().Add(c.manager.Config().Server.HTTP.ReadHeaderTimeout)
		case 2:
			deadline = time.Now().Add(c.manager.Config().Server.HTTP.IdleTimeout)
		}
	}
	return c.Conn.SetReadDeadline(deadline)
}
