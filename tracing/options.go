package tracing

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type options struct {
	tracerProvider trace.TracerProvider
	propagator     propagation.TextMapPropagator
	kind           trace.SpanKind
	tracerName     string
	spanName       string
}

type Option func(*options)

func WithTracerName(tracerName string) Option {
	return func(opts *options) {
		opts.tracerName = tracerName
	}
}

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(opts *options) {
		opts.propagator = propagator
	}
}

func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(opts *options) {
		opts.tracerProvider = provider
	}
}

func WithGlobalTracerProvider() Option {
	return func(opts *options) {
		opts.tracerProvider = otel.GetTracerProvider()
	}
}

func WithGlobalPropagator() Option {
	return func(opts *options) {
		opts.propagator = otel.GetTextMapPropagator()
	}
}
