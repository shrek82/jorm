package middleware

import (
	"context"

	"github.com/shrek82/jorm/core"
)

// TracingMiddleware adds tracing information to the query context.
// It extracts information like Request ID or User IP from the context
// and attaches it to the query execution logger.
type TracingMiddleware struct{}

func NewTracing() *TracingMiddleware {
	return &TracingMiddleware{}
}

func (m *TracingMiddleware) Name() string {
	return "Tracing"
}

func (m *TracingMiddleware) Init(db *core.DB) error {
	return nil
}

func (m *TracingMiddleware) Shutdown() error {
	return nil
}

func (m *TracingMiddleware) Process(ctx context.Context, query *core.Query, next core.QueryFunc) (*core.Result, error) {
	// Extract info from context
	// This assumes the user has put something in context using common keys.
	// You might want to make these keys configurable.

	fields := make(map[string]any)

	if reqID := ctx.Value("request_id"); reqID != nil {
		fields["request_id"] = reqID
	}
	if userIP := ctx.Value("user_ip"); userIP != nil {
		fields["user_ip"] = userIP
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}

	if len(fields) > 0 {
		query.WithFields(fields)
	}

	return next(ctx, query)
}
