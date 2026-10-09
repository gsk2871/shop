// Package bootstrap owns the startup sequence for the single service.
package bootstrap

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"peptide-site/internal/conf"
	"peptide-site/internal/data"
)

// Run loads settings, runs maintenance commands or starts the wired server.
func Run(version string, factory func(conf.Config) (*khttp.Server, func(), error)) {
	configPath := flag.String("conf", "", "YAML configuration path (default configs/config.local.yaml)")
	migrate := flag.Bool("migrate", false, "apply database schema")
	initAdmin := flag.Bool("init-admin", false, "create administrator from ADMIN_USERNAME and ADMIN_PASSWORD")
	flag.Parse()
	var config conf.Config
	var err error
	if *configPath == "" {
		config, err = conf.Load()
	} else {
		config, err = conf.LoadFile(*configPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	if *migrate || *initAdmin {
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
		if *initAdmin {
			if err := data.InitAdmin(context.Background(), d, os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD")); err != nil {
				log.Fatal(err)
			}
		}
		return
	}
	server, cleanup, err := factory(config)
	if err != nil {
		log.Fatal("backend initialization failed; check database connectivity and configuration")
	}
	defer cleanup()
	app := kratos.New(kratos.Name(config.App.Name), kratos.Version(version), kratos.Server(server))
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
