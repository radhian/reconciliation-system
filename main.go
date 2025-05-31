package main

import (
	"os"

	"github.com/radhian/reconciliation-system/controllers"
)

func main() {
	app := controllers.App{}
	app.Initialize(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"))

	app.RunServer()
}
