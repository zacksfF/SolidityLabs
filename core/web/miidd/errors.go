package miidd

import (
	"context"
	"errors"
	"net/http"

	"github.com/zacksfF/FullStack-Blockchain/core/validate"
	"github.com/zacksfF/FullStack-Blockchain/core/web/errs"
	"github.com/zacksfF/FullStack-Blockchain/web2"
	"go.uber.org/zap"
)

// Errors handles errors coming out of the call chain. It detects normal
// application errors which are used to respond to the client in a uniform way.
// Unexpected errors (status >= 500) are logged.
func Errors(log *zap.SugaredLogger) web2.Middleware {

	// This is the actual middleware function to be executed.
	m := func(handler web2.Handler) web2.Handler {

		// Create the handler that will be attached in the middleware chain.
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			// If the context is missing this value, request the service
			// to be shutdown gracefully.
			v, err := web2.GetValues(ctx)
			if err != nil {
				return web2.NewShutdownError("web value missing from context")
			}

			// Run the next handler and catch any propagated error.
			if err := handler(ctx, w, r); err != nil {

				// Log the error.
				log.Errorw("ERROR", "traceid", v.TraceID, "ERROR", err)

				// Build out the error response.
				var er errs.Response
				var status int
				switch {
				case validate.IsFieldErrors(err):
					fieldErrors := validate.GetFieldErrors(err)
					er = errs.Response{
						Error:  "data validation error",
						Fields: fieldErrors.Fields(),
					}
					status = http.StatusBadRequest

				case errs.IsTrusted(err):
					reqErr := errs.GetTrusted(err)
					er = errs.Response{
						Error: reqErr.Error(),
					}
					status = reqErr.Status

				default:
					er = errs.Response{
						Error: http.StatusText(http.StatusInternalServerError),
					}
					status = http.StatusInternalServerError
				}

				// Respond with the error back to the client.
				if err := web2.Respond(ctx, w, er, status); err != nil {

					// If we get this error, it means the event handler
					// has completed.
					if !errors.Is(err, http.ErrHijacked) {
						return err
					}
				}

				// If we receive the shutdown err we need to return it
				// back to the base handler to shut down the service.
				if ok := web2.IsShutdown(err); ok {
					return err
				}
			}

			// The error has been handled so we can stop propagating it.
			return nil
		}

		return h
	}

	return m
}
