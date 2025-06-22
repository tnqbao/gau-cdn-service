package main

import (
	"github.com/joho/godotenv"
	"github.com/tnqbao/gau-cdn-service/config"
	"github.com/tnqbao/gau-cdn-service/controller"
	"github.com/tnqbao/gau-cdn-service/infra"
	"github.com/tnqbao/gau-cdn-service/routes"
	"log"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// Initialize configuration and infrastructure
	newConfig := config.NewConfig()
	newInfra := infra.InitInfra(newConfig)

	// Initialize controller with the new configuration and infrastructure
	ctrl := controller.NewController(newConfig, newInfra)

	router := routes.SetupRouter(ctrl)
	router.Run(":8080")
}
