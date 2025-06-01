package handler

import (
	"context"
	"errors"
)

func (h *ReconciliationHandler) ReconciliationExecution(ctx context.Context) error {
	acquired, logID, err := h.Usecase.TryAcquireLock(ctx)
	if err != nil {
		return err
	}

	if !acquired {
		return errors.New("no process handled")
	}

	defer h.Usecase.UnlockProcess(ctx, logID)

	err = h.Usecase.ProcessReconciliationJob(ctx, logID)
	if err != nil {
		return err
	}

	return nil
}
