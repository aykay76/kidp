package controller

import (
	"context"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatusWithFallback tries to update the status subresource, and if the
// underlying client doesn't support the status subresource (fake client may
// return NotFound), falls back to a full Update. The provided logger is used
// for error messages.
func UpdateStatusWithFallback(ctx context.Context, c client.Client, obj client.Object, logger logr.Logger) error {
	if err := c.Status().Update(ctx, obj); err != nil {
		logger.Error(err, "Status().Update failed")
		if apierrors.IsNotFound(err) {
			// Fallback to full update for clients which do not support status subresource
			if uerr := c.Update(ctx, obj); uerr != nil {
				logger.Error(uerr, "Fallback Update after Status().Update failed")
				return uerr
			}
			return nil
		}
		return err
	}
	return nil
}
