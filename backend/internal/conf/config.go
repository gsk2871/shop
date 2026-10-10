// Package conf loads the single service's runtime configuration.
package conf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the private local configuration, never the tracked template.
const DefaultPath = "configs/config.local.yaml"

// Config groups settings by infrastructure responsibility.
type Config struct {
	App     App     `yaml:"app"`
	Server  Server  `yaml:"server"`
	Data    Data    `yaml:"data"`
	Auth    Auth    `yaml:"auth"`
	Reload  Reload  `yaml:"reload"`
	Log     Log     `yaml:"log"`
	Contact Contact `yaml:"contact"`
}

type Contact struct {
	GeoIPDatabasePath string `yaml:"geoip_database_path"`
}

type Reload struct {
	Interval time.Duration `yaml:"interval"`
}
type Log struct {
	Level string `yaml:"level"`
}

// App identifies the one deployed service.
type App struct {
	Name string `yaml:"name"`
}

// Server configures the single HTTP listener for public and admin APIs.
type Server struct {
	HTTP HTTP `yaml:"http"`
}

// HTTP contains request and connection timeouts.
type HTTP struct {
	Addr              string        `yaml:"addr"`
	Timeout           time.Duration `yaml:"timeout"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
}

// Data contains database and public file storage settings.
type Data struct {
	Database Database `yaml:"database"`
	Storage  Storage  `yaml:"storage"`
}

// Database configures PostgreSQL and its connection pool.
type Database struct {
	DSN              string        `yaml:"dsn"`
	MaxOpenConns     int           `yaml:"max_open_conns"`
	MaxIdleConns     int           `yaml:"max_idle_conns"`
	ConnMaxLifetime  time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime  time.Duration `yaml:"conn_max_idle_time"`
	ConnectTimeout   time.Duration `yaml:"connect_timeout"`
	StatementTimeout time.Duration `yaml:"statement_timeout"`
	LockTimeout      time.Duration `yaml:"lock_timeout"`
	LogSQL           bool          `yaml:"log_sql"`
	SlowSQLMS        int           `yaml:"slow_sql_ms"`
}

// Storage configures the local upload directory.
type Storage struct {
	UploadDir        string        `yaml:"upload_dir"`
	MaxImageBytes    int64         `yaml:"max_image_bytes"`
	MaxVideoBytes    int64         `yaml:"max_video_bytes"`
	MaxDocumentBytes int64         `yaml:"max_document_bytes"`
	AllowedTypes     []string      `yaml:"allowed_types"`
	CleanupInterval  time.Duration `yaml:"cleanup_interval"`
}

type TOTP struct {
	Enabled     bool              `yaml:"enabled"`
	Issuer      string            `yaml:"issuer"`
	ActiveKeyID string            `yaml:"active_key_id"`
	Keys        map[string]string `yaml:"keys"`
}

// Auth contains administrator session cookie settings.
type Auth struct {
	CookieSecure   bool          `yaml:"cookie_secure"`
	SessionTTL     time.Duration `yaml:"session_ttl"`
	OnboardingTTL  time.Duration `yaml:"onboarding_ttl"`
	MFATicketTTL   time.Duration `yaml:"mfa_ticket_ttl"`
	MFAMaxAttempts int           `yaml:"mfa_max_attempts"`
	LoginLimit     int           `yaml:"login_limit"`
	TrustedProxies []string      `yaml:"trusted_proxies"`
	TOTP           TOTP          `yaml:"totp"`
}

func defaults() Config {
	return Config{
		Reload: Reload{Interval: 5 * time.Second},
		Log:    Log{Level: "info"},
		App:    App{Name: "peptide-site"},
		Server: Server{HTTP: HTTP{Addr: "127.0.0.1:8080", Timeout: 30 * time.Second, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 120 * time.Second}},
		Data:   Data{Database: Database{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: 30 * time.Minute, ConnMaxIdleTime: 15 * time.Minute, ConnectTimeout: 10 * time.Second, StatementTimeout: 5 * time.Second, LockTimeout: 3 * time.Second, SlowSQLMS: 200}, Storage: Storage{UploadDir: "./storage/uploads", MaxImageBytes: 20 << 20, MaxVideoBytes: 200 << 20, MaxDocumentBytes: 20 << 20, AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif", "video/mp4", "video/webm", "application/pdf"}, CleanupInterval: time.Minute}},
		Auth:   Auth{CookieSecure: true, SessionTTL: 8 * time.Hour, OnboardingTTL: 15 * time.Minute, MFATicketTTL: 5 * time.Minute, MFAMaxAttempts: 5, LoginLimit: 5, TOTP: TOTP{Enabled: true, Issuer: "Research Supply Admin"}},
	}
}

// Load permits environment-only deployment if the default local file is absent.
func Load() (Config, error) { return load(DefaultPath, true) }

// LoadFile requires the explicitly selected configuration file to exist.
func LoadFile(path string) (Config, error) { return load(path, false) }

// Loader captures environment overrides once. File reload never reads a changed process environment.
type Loader struct {
	overrides map[string]string
	mu        sync.Mutex
	sources   map[string]string
}

func NewLoader() *Loader {
	l := &Loader{overrides: map[string]string{}}
	for _, name := range []string{"HTTP_ADDRESS", "DATABASE_DSN", "UPLOAD_DIR", "COOKIE_SECURE"} {
		if v, ok := os.LookupEnv(name); ok {
			l.overrides[name] = v
		}
	}
	return l
}
func (l *Loader) Sources() map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := map[string]string{}
	for k, v := range l.sources {
		r[k] = v
	}
	return r
}
func configLeaves(v reflect.Value, prefix string, result map[string]string) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		path := field.Tag.Get("yaml")
		if prefix != "" {
			path = prefix + "." + path
		}
		value := v.Field(i)
		if value.Kind() == reflect.Struct {
			configLeaves(value, path, result)
		} else {
			result[path] = "default"
		}
	}
}
func yamlLeaves(values map[string]any, prefix string, result map[string]string) {
	for name, value := range values {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if child, ok := value.(map[string]any); ok {
			yamlLeaves(child, path, result)
		} else {
			result[path] = "yaml"
		}
	}
}
func load(path string, optional bool) (Config, error) { return NewLoader().Load(path, optional) }
func (l *Loader) Load(path string, optional bool) (Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if optional && errors.Is(err, os.ErrNotExist) {
			return l.Decode(nil)
		}
		return Config{}, fmt.Errorf("cannot read configuration file %q", path)
	}
	return l.Decode(body)
}
func (l *Loader) Decode(body []byte) (Config, error) {
	c := defaults()
	if len(body) > 0 {
		d := yaml.NewDecoder(bytes.NewReader(body))
		d.KnownFields(true)
		if err := d.Decode(&c); err != nil {
			return Config{}, errors.New("invalid YAML configuration")
		}
		if err := d.Decode(new(any)); err != io.EOF {
			return Config{}, errors.New("configuration must contain one YAML document")
		}
	}
	for name, target := range map[string]*string{"HTTP_ADDRESS": &c.Server.HTTP.Addr, "DATABASE_DSN": &c.Data.Database.DSN, "UPLOAD_DIR": &c.Data.Storage.UploadDir} {
		if v := l.overrides[name]; v != "" {
			*target = v
		}
	}
	if v, ok := l.overrides["COOKIE_SECURE"]; ok {
		var err error
		c.Auth.CookieSecure, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, errors.New("COOKIE_SECURE must be true or false")
		}
	}
	if err := Validate(c); err != nil {
		return Config{}, err
	}
	sources := map[string]string{}
	configLeaves(reflect.ValueOf(c), "", sources)
	if len(body) > 0 {
		var values map[string]any
		_ = yaml.Unmarshal(body, &values)
		yamlLeaves(values, "", sources)
	}
	for name, path := range map[string]string{"HTTP_ADDRESS": "server.http.addr", "DATABASE_DSN": "data.database.dsn", "UPLOAD_DIR": "data.storage.upload_dir", "COOKIE_SECURE": "auth.cookie_secure"} {
		if v, ok := l.overrides[name]; ok && (v != "" || name == "COOKIE_SECURE") {
			sources[path] = "environment:" + name
		}
	}
	l.mu.Lock()
	l.sources = sources
	l.mu.Unlock()
	return c, nil
}

// Validate rejects invalid settings without including database credentials.
func Validate(c Config) error {
	storage := c.Data.Storage
	if storage.MaxImageBytes <= 0 || storage.MaxVideoBytes <= 0 || storage.MaxDocumentBytes <= 0 || storage.MaxImageBytes > 1<<40 || storage.MaxVideoBytes > 1<<40 || storage.MaxDocumentBytes > 1<<40 || storage.CleanupInterval < 100*time.Millisecond || storage.CleanupInterval > time.Hour || len(storage.AllowedTypes) == 0 {
		return errors.New("invalid storage upload policy")
	}
	allowed := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true, "video/mp4": true, "video/webm": true, "application/pdf": true}
	seen := map[string]bool{}
	for _, mime := range storage.AllowedTypes {
		if !allowed[mime] || seen[mime] {
			return errors.New("invalid or duplicate storage type")
		}
		seen[mime] = true
	}

	if c.Auth.SessionTTL <= 0 || c.Auth.SessionTTL > 24*time.Hour || c.Auth.OnboardingTTL <= 0 || c.Auth.OnboardingTTL > time.Hour || c.Auth.MFATicketTTL <= 0 || c.Auth.MFATicketTTL > 15*time.Minute || c.Auth.MFAMaxAttempts < 1 || c.Auth.MFAMaxAttempts > 20 || c.Auth.LoginLimit < 1 || c.Auth.TOTP.Issuer == "" {
		return errors.New("invalid authentication policy")
	}
	for _, p := range c.Auth.TrustedProxies {
		if _, _, err := net.ParseCIDR(p); err != nil {
			return errors.New("auth.trusted_proxies requires CIDR prefixes")
		}
	}

	if c.Reload.Interval < 100*time.Millisecond || c.Reload.Interval > time.Hour {
		return errors.New("reload.interval must be between 100ms and 1h")
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("log.level must be debug, info, warn or error")
	}
	if strings.TrimSpace(c.App.Name) == "" {
		return errors.New("app.name is required")
	}
	if strings.TrimSpace(c.Data.Database.DSN) == "" {
		return errors.New("data.database.dsn or DATABASE_DSN is required; copy configs/config.local.example.yaml to configs/config.local.yaml")
	}
	if strings.TrimSpace(c.Data.Storage.UploadDir) == "" {
		return errors.New("data.storage.upload_dir is required")
	}
	host, port, err := net.SplitHostPort(c.Server.HTTP.Addr)
	if err != nil || strings.ContainsAny(host, " \t\r\n") {
		return errors.New("server.http.addr must be host:port")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return errors.New("server.http.addr port must be between 1 and 65535")
	}
	h := c.Server.HTTP
	if h.Timeout <= 0 || h.ReadHeaderTimeout <= 0 || h.ReadTimeout <= 0 || h.WriteTimeout <= 0 || h.IdleTimeout <= 0 {
		return errors.New("server.http timeouts must be positive")
	}
	if h.WriteTimeout <= h.Timeout || h.WriteTimeout <= h.ReadTimeout {
		return errors.New("server.http.write_timeout must exceed timeout and read_timeout")
	}
	d := c.Data.Database
	if d.MaxOpenConns <= 0 || d.MaxIdleConns < 0 || d.MaxIdleConns > d.MaxOpenConns || d.ConnMaxLifetime <= 0 {
		return errors.New("invalid data.database connection pool settings")
	}
	if d.ConnMaxIdleTime <= 0 || d.ConnectTimeout < time.Second || d.ConnectTimeout > 30*time.Second || d.StatementTimeout < time.Millisecond || d.StatementTimeout > 30*time.Second || d.LockTimeout < time.Millisecond || d.LockTimeout.Milliseconds() >= d.StatementTimeout.Milliseconds() || d.SlowSQLMS < 1 || d.SlowSQLMS > 30000 {
		return errors.New("invalid data.database timeout or SQL logging settings")
	}
	return nil
}
