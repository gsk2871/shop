// Package bootstrap owns the startup sequence for the single service.
package bootstrap

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/go-kratos/kratos/v2"
	"peptide-site/internal/pkg/configreload"

	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	adminData "peptide-site/internal/data/admin"
)

// Run loads settings, runs maintenance commands or starts the wired server.
func Run(version string, factory func(conf.Config) (*configreload.Resource, func(), error)) {
	configPath := flag.String("conf", "", "YAML configuration path (default configs/config.local.yaml)")
	migrate := flag.Bool("migrate", false, "apply database schema")
	initAdmin := flag.Bool("init-admin", false, "alias for -seed")
	recoverUser := flag.String("recover-totp-user", "", "server-only MFA recovery username")
	recoverReason := flag.String("recover-reason", "", "required reason for MFA recovery")
	seed := flag.Bool("seed", false, "initialize permissions, roles and first administrator")
	seedUsername := flag.String("seed-admin-username", "admin", "显式初始化时的超管用户名")
	flag.Parse()
	explicitConfig := *configPath != ""
	loader := conf.NewLoader()
	if *configPath == "" {
		*configPath = conf.DefaultPath
	}
	var config conf.Config
	var err error
	config, err = loader.Load(*configPath, !explicitConfig)
	if err != nil {
		log.Fatal(err)
	}
	// 维护命令和启动准备阶段也使用配置中的日志等级。
	level := slog.LevelInfo
	_ = level.UnmarshalText([]byte(config.Log.Level))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
	if *migrate || *initAdmin || *seed || *recoverUser != "" {
		d, cleanup, err := data.NewData(config)
		if err != nil {
			log.Fatal("database unavailable")
		}
		defer cleanup()
		if *migrate {
			if err := data.Migrate(d); err != nil {
				log.Fatal("schema migration failed")
			}
		}
		if *initAdmin || *seed {
			if err := adminData.Seed(context.Background(), d, adminData.SeedOptions{Username: *seedUsername, Password: os.Getenv("SEED_ADMIN_PASSWORD")}); err != nil {
				log.Fatal(err)
			}
		}
		if *recoverUser != "" {
			repo, err := adminData.NewRepo(d, config)
			if err != nil {
				log.Fatal("MFA configuration invalid")
			}
			if err := repo.Recover(context.Background(), *recoverUser, *recoverReason); err != nil {
				log.Fatal("MFA recovery failed")
			}
		}
		return
	}

	manager, err := configreload.New(context.Background(), *configPath, config, loader, func(ctx context.Context, c conf.Config) (*configreload.Resource, error) {
		resource, cleanup, err := factory(c)
		if err != nil {
			return nil, err
		}
		resourceClose := resource.Close
		resource.Close = func() {
			if resourceClose != nil {
				resourceClose()
			}
			cleanup()
		}
		return resource, nil
	})
	if err != nil {
		log.Fatal("backend resource preparation failed; check schema, database and storage")
	}
	manager.ConfigureLogging()
	server := configreload.NewHTTP(manager)
	app := kratos.New(kratos.Name(config.App.Name), kratos.Version(version), kratos.Server(server))
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
