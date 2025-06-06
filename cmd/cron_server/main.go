package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	"github.com/radhian/reconciliation-system/consts"
	"github.com/radhian/reconciliation-system/handler"
	"github.com/radhian/reconciliation-system/infra/db/dao"
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
			if err.Error() == consts.NoProcessHandled {
				log.Printf("[Worker %d] %s", workerID, err.Error())
			} else {
				log.Printf("[Worker %d] error: %s", workerID, err.Error())
			}
		} else {
			log.Printf("[Worker %d] success", workerID)
		}

		time.Sleep(cfg.Interval)
	}
}

type AppConfig struct {
	BatchSize     int
	WorkerNumber  int
	IntervalInSec int
}

func NewAppConfig() (*AppConfig, error) {
	batchSizeStr := os.Getenv("BATCH_SIZE")
	workerStr := os.Getenv("NUMBER_OF_WORKER")
	intervalStr := os.Getenv("INTERVAL_IN_SEC")

	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BATCH_SIZE: %v", err)
	}

	numWorker, err := strconv.Atoi(workerStr)
	if err != nil {
		return nil, fmt.Errorf("invalid NUMBER_OF_WORKER: %v", err)
	}

	intervalSec, err := strconv.Atoi(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid INTERVAL_IN_SEC: %v", err)
	}

	cfg := &AppConfig{
		BatchSize:     batchSize,
		WorkerNumber:  numWorker,
		IntervalInSec: intervalSec,
	}

	return cfg, nil
}

type App struct {
	DB     *gorm.DB
	Locker *locker.Locker
	Config *AppConfig
}

func (a *App) startCronWorker(cfg CronWorkerConfig) {
	var wg sync.WaitGroup

	batchSize := int64(consts.DefaultBatchSize)
	if a.Config != nil {
		batchSize = int64(a.Config.BatchSize)
	}

	reconciliationDao := dao.NewDaoMethod(a.DB)
	reconciliationUc := reconciliationUsecase.NewReconciliationUsecase(reconciliationDao, a.Locker, batchSize)
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

	a.Config, err = NewAppConfig()
	if err != nil {
		fmt.Printf("\n Cannot get config from .env, use default, err %s", err.Error())
	}
}

func (a *App) RunServer() {
	workerNumber := consts.DefaultWorkerNumber
	intervalInSec := consts.DefaultIntervalInSec
	if a.Config != nil {
		workerNumber = a.Config.WorkerNumber
		intervalInSec = a.Config.IntervalInSec
	}

	a.startCronWorker(CronWorkerConfig{
		Workers:  workerNumber,
		Interval: time.Duration(intervalInSec) * time.Second,
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
