//go:build wireinject

package main

import (
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/wire"
	"peptide-site/internal/biz"
	bizcms "peptide-site/internal/biz/cms"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	datacms "peptide-site/internal/data/cms"
	"peptide-site/internal/server"
	"peptide-site/internal/service"
	servicecms "peptide-site/internal/service/cms"
)

func initializeServer(c conf.Config) (*khttp.Server, func(), error) {
	panic(wire.Build(data.NewData, wire.Bind(new(biz.Checker), new(*data.Data)), datacms.NewRepo, wire.Bind(new(bizcms.Repository), new(*datacms.Repo)), servicecms.NewHandler, biz.NewSystem, service.NewSystem, server.NewHTTPServer))
}
