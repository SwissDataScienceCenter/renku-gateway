package main

import (
	"context"

	sentryotel "github.com/getsentry/sentry-go/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// initOpenTelemetry initializes OpenTelemetry with a trace provider
func initOpenTelemetry(environment string, sampleRate float64) (*sdktrace.TracerProvider, error) {
	// Create a resource with service information
	resource, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("renku-gateway"),
			semconv.DeploymentEnvironment(environment),
			semconv.TelemetrySDKLanguageGo,
		),
	)
	if err != nil {
		return nil, err
	}

	var sampler sdktrace.Sampler
	if sampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if sampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(sampleRate)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tracerProvider)

	// Use a composite propagator to support both OpenTelemetry and Sentry headers
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{}, // TODO: Is this required or it's covered by sentryotel?
		sentryotel.NewSentryPropagator(),
	))

	return tracerProvider, nil
}

// shutdownOpenTelemetry shuts down the trace provider
func shutdownOpenTelemetry(ctx context.Context, tracerProvider *sdktrace.TracerProvider) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}
