//go:build wireinject

package main

import (
	"github.com/google/wire"
	"peptide-site/internal/biz"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/server"
	"peptide-site/internal/service"
	servicecms "peptide-site/internal/service/cms"
)

func initializeServer(c conf.Config) (*configreload.Resource, func(), error) {
	panic(wire.Build(data.NewData, wire.Bind(new(biz.Checker), new(*data.Data)), servicecms.NewHandler, biz.NewSystem, service.NewSystem, server.NewResources))
}
