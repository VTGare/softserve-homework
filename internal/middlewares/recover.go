package middlewares

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

//Recover is a panic recovery middleware.
func Recover(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Errorw("panic recovery", "error", err, "traceback", debug.Stack())
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
