package handler

import (
	"context"
)

func (h *ReconciliationHandler) ReconciliationExecution(ctx context.Context) error {
	acquired, err := h.Usecase.TryAcquireLock(ctx)
	if err != nil {
		return err
	}

	if !acquired {
		//TODO: add log, system is busy
		return nil
	}

	err = h.Usecase.ProcessReconciliationJob(ctx)
	if err != nil {
		return err
	}

	return nil
}
