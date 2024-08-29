package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartTrace creates a span using the default tracer instance.
func StartTrace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	attributes := trace.WithInstrumentationAttributes(attribute.String("scope", "joy"))

	return otel.Tracer("", attributes).Start(ctx, spanName, opts...)
}
