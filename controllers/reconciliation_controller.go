package controllers

import (
	"github.com/radhian/reconciliation-system/handler"

	"github.com/gorilla/mux"
)

func RegisterReconciliationRoutes(router *mux.Router, h *handler.ReconciliationHandler) {
	router.HandleFunc("/process_reconciliation", h.ProcessReconciliation).Methods("POST")
	router.HandleFunc("/get_result", h.GetResult).Methods("GET")
}
