// Package conf loads the single service's runtime configuration.
package conf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the private local configuration, never the tracked template.
const DefaultPath = "configs/config.local.yaml"

// Config groups settings by infrastructure responsibility.
type Config struct {
	App    App    `yaml:"app"`
	Server Server `yaml:"server"`
	Data   Data   `yaml:"data"`
	Auth   Auth   `yaml:"auth"`
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
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// Storage configures the local upload directory.
type Storage struct {
	UploadDir string `yaml:"upload_dir"`
}

// Auth contains administrator session cookie settings.
type Auth struct {
	CookieSecure bool `yaml:"cookie_secure"`
}

func defaults() Config {
	return Config{
		App:    App{Name: "peptide-site"},
		Server: Server{HTTP: HTTP{Addr: "127.0.0.1:8080", Timeout: 30 * time.Second, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 120 * time.Second}},
		Data:   Data{Database: Database{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: 30 * time.Minute}, Storage: Storage{UploadDir: "./storage/uploads"}},
		Auth:   Auth{CookieSecure: true},
	}
}

// Load permits environment-only deployment if the default local file is absent.
func Load() (Config, error) { return load(DefaultPath, true) }

// LoadFile requires the explicitly selected configuration file to exist.
func LoadFile(path string) (Config, error) { return load(path, false) }

func load(path string, optional bool) (Config, error) {
	c := defaults()
	body, err := os.ReadFile(path)
	if err != nil && !(optional && errors.Is(err, os.ErrNotExist)) {
		return Config{}, fmt.Errorf("cannot read configuration file %q", path)
	}
	if err == nil {
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
		if v := os.Getenv(name); v != "" {
			*target = v
		}
	}
	if v, ok := os.LookupEnv("COOKIE_SECURE"); ok {
		c.Auth.CookieSecure, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, errors.New("COOKIE_SECURE must be true or false")
		}
	}
	if err := Validate(c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate rejects invalid settings without including database credentials.
func Validate(c Config) error {
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
	return nil
}
