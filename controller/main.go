package controller

import (
	"github.com/tnqbao/gau-cdn-service/config"
	"github.com/tnqbao/gau-cdn-service/infra"
	"github.com/tnqbao/gau-cdn-service/provider"
	"github.com/tnqbao/gau-cdn-service/repository"
)

type Controller struct {
	Config     *config.Config
	Infra      *infra.Infra
	Repository *repository.Repository
	Provider   *provider.Provider
}

func NewController(cfg *config.Config, infra *infra.Infra) *Controller {
	newRepository := repository.InitRepository(infra, cfg.EnvConfig)
	provide := provider.InitProvider(cfg.EnvConfig)
	return &Controller{
		Config:     cfg,
		Infra:      infra,
		Repository: newRepository,
		Provider:   provide,
	}
}
