package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	"github.com/radhian/reconciliation-system/handler"
	"github.com/radhian/reconciliation-system/infra/locker"
	reconciliationUsecase "github.com/radhian/reconciliation-system/usecase/reconciliation"
)

type CronWorkerConfig struct {
	Interval time.Duration
	Workers  int
}

func (cfg CronWorkerConfig) startReconcileExecutorWorker(h *handler.ReconciliationHandler, workerID int) {
	for {
		ctx := context.Background()
		err := h.ReconciliationExecution(ctx)
		if err != nil {
			log.Printf("[Worker %d] error: %s", workerID, err.Error())
		} else {
			log.Printf("[Worker %d] success", workerID)
		}

		time.Sleep(cfg.Interval)
	}
}

type App struct {
	DB     *gorm.DB
	Locker *locker.Locker
}

func (a *App) startCronWorker(cfg CronWorkerConfig) {
	var wg sync.WaitGroup

	reconciliationUc := reconciliationUsecase.NewReconciliationUsecase(a.DB, a.Locker)
	h := handler.NewReconciliationHandler(reconciliationUc)

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			log.Printf("spawn [Worker %d]", workerID)
			cfg.startReconcileExecutorWorker(h, workerID)
		}(i + 1)
	}
	wg.Wait()
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

	a.Locker = locker.New()
}

func (a *App) RunServer() {
	a.startCronWorker(CronWorkerConfig{
		Workers:  1,
		Interval: 2 * time.Second,
	})
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
