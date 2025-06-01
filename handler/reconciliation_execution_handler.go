package handler

import (
	"context"
	"errors"

	"github.com/radhian/reconciliation-system/consts"
)

func (h *ReconciliationHandler) ReconciliationExecution(ctx context.Context) error {
	acquired, logID, err := h.Usecase.TryAcquireLock(ctx)
	if err != nil {
		return err
	}

	if !acquired {
		return errors.New(consts.NoProcessHandled)
	}

	defer h.Usecase.UnlockProcess(ctx, logID)

	err = h.Usecase.ProcessReconciliationJob(ctx, logID)
	if err != nil {
		return err
	}

	return nil
}
