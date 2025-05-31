package handler

import (
	"context"
	"log"
)

func (h *ReconciliationHandler) ReconciliationExecution(ctx context.Context) error {
	acquired, logID, err := h.Usecase.TryAcquireLock(ctx)
	if err != nil {
		return err
	}

	if !acquired {
		log.Printf("[INFO] All process is busy")
		return nil
	}

	defer h.Usecase.UnlockProcess(ctx, logID)

	err = h.Usecase.ProcessReconciliationJob(ctx, logID)
	if err != nil {
		return err
	}

	return nil
}
