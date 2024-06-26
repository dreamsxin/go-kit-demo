package addservice

import (
	"context"

	"github.com/dreamsxin/go-kit/log"
)

// Middleware describes a service (as opposed to endpoint) middleware.
type Middleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a service Middleware.
func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger *log.Logger
	next   Service
}

func (mw loggingMiddleware) Sum(ctx context.Context, a, b int) (v int, err error) {
	defer func() {
		mw.logger.Sugar().Debugln("method", "Sum", "a", a, "b", b, "v", v, "err", err)
	}()
	return mw.next.Sum(ctx, a, b)
}

func (mw loggingMiddleware) Concat(ctx context.Context, a, b string) (v string, err error) {
	defer func() {
		mw.logger.Sugar().Debugln("method", "Concat", "a", a, "b", b, "v", v, "err", err)
	}()
	return mw.next.Concat(ctx, a, b)
}
