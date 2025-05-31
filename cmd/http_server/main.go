package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	"github.com/radhian/reconciliation-system/handler"
	"github.com/radhian/reconciliation-system/infra/db/model"
	"github.com/radhian/reconciliation-system/middlewares"
	reconciliationUsecase "github.com/radhian/reconciliation-system/usecase/reconciliation"
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
		&model.ReconciliationProcessLog{},
		&model.ReconciliationProcessLogAsset{},
	) //database migration

	a.Router = mux.NewRouter().StrictSlash(true)
	a.initializeRoutes()
}

func RegisterReconciliationRoutes(router *mux.Router, h *handler.ReconciliationHandler) {
	router.HandleFunc("/process_reconciliation", h.ProcessReconciliation).Methods("POST")
	router.HandleFunc("/get_result", h.GetResult).Methods("GET")
}

func (a *App) initializeRoutes() {
	a.Router.Use(middlewares.SetContentTypeMiddleware)
	reconciliationUc := reconciliationUsecase.NewReconciliationUsecase(a.DB, nil)
	handler := handler.NewReconciliationHandler(reconciliationUc)
	RegisterReconciliationRoutes(a.Router, handler)
}

func (a *App) RunServer() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("\nServer starting on port %v", port)
	log.Fatal(http.ListenAndServe(":"+port, a.Router))
}

func main() {
	app := App{}
	app.Initialize(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"))

	app.RunServer()
}
