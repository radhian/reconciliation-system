package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Lgdev07/reconciliation_system/middlewares"
	"github.com/Lgdev07/reconciliation_system/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
)

type App struct {
	DB     *gorm.DB
	Router *mux.Router
}

func (a *App) Initialize(DbHost, DbPort, DbUser, DbName, DbPassword string) {
	var err error
	DBURI := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", DbHost, DbPort, DbUser, DbName, DbPassword)
	fmt.Printf("DB Config - Host: %q, Port: %q, User: %q, Name: %q, Password: %q\n", DbHost, DbPort, DbUser, DbName, DbPassword)

	a.DB, err = gorm.Open("postgres", DBURI)
	if err != nil {
		fmt.Printf("\n Cannot connect to database %s", DbName)
		log.Fatal("This is the error:", err)
	} else {
		fmt.Printf("We are connected to the database %s", DbName)
	}

	a.DB.Debug().AutoMigrate(
		&models.ReconciliationProcessLog{},
		&models.ReconciliationProcessLogAsset{},
	) //database migration

	a.Router = mux.NewRouter().StrictSlash(true)
	a.initializeRoutes()
}

func (a *App) initializeRoutes() {
	a.Router.Use(middlewares.SetContentTypeMiddleware)
}

func (a *App) RunServer() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("\nServer starting on port %v", port)
	log.Fatal(http.ListenAndServe(":"+port, a.Router))
}
