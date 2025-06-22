package controller

import (
	"github.com/tnqbao/gau-cdn-service/config"
	"github.com/tnqbao/gau-cdn-service/infra"
	"github.com/tnqbao/gau-cdn-service/repository"
)

type Controller struct {
	Config     *config.Config
	Infra      *infra.Infra
	Repository *repository.Repository
}

func NewController(cfg *config.Config, infra *infra.Infra) *Controller {
	newRepository := repository.InitRepository(infra)
	return &Controller{
		Config:     cfg,
		Infra:      infra,
		Repository: newRepository,
	}
}
