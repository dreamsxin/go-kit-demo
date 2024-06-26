package addendpoint

import (
	"context"
	"time"

	"github.com/dreamsxin/go-kit/endpoint"
	"github.com/dreamsxin/go-kit/log"
)

// LoggingMiddleware returns an endpoint middleware that logs the
// duration of each invocation, and the resulting error, if any.
func LoggingMiddleware(logger *log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				logger.Sugar().Errorln("transport_error", err, "took", time.Since(begin))
			}(time.Now())
			return next(ctx, request)

		}
	}
}
