# OpenTelemetry Integration with Sentry

## Overview

The gateway now uses **OpenTelemetry (OTel)** as the primary tracing system, with Sentry configured to use OTel-generated trace IDs. This provides a vendor-neutral, standards-based approach to distributed tracing while maintaining full Sentry functionality.

## Architecture

```
Incoming Request
    ↓
Echo RequestID Middleware
    ↓
OTel Echo Middleware (generates W3C trace context)
    ↓
Sentry Middleware (uses OTel trace IDs via span processor)
    ↓
sentryHeaderInjector (injects both traceparent and sentry-trace headers)
    ↓
Your application middlewares
    ↓
Outgoing requests (with W3C + Sentry headers)
```

## Key Components

### 1. OpenTelemetry Initialization (`cmd/gateway/otel.go`)
- Initializes OTel trace provider with service name and environment
- Configures sampling rate to match Sentry's sample rate
- Adds Sentry span processor to bridge OTel spans to Sentry
- Sets up W3C Trace Context and Sentry propagators

### 2. Middleware Chain (`cmd/gateway/main.go`)
- **Order matters**: OTel middleware must come before Sentry middleware
- Sentry is initialized first (required by span processor)
- OTel trace provider is initialized with Sentry integration
- Both middlewares are added to the Echo server

### 3. Header Injection (`cmd/gateway/middlewares.go`)
- `sentryHeaderInjector` now injects both:
  - **W3C headers**: `traceparent`, `tracestate` (from OTel)
  - **Sentry headers**: `sentry-trace`, `baggage` (for backward compatibility)
- Downstream services can use either header format

### 4. Trace ID Extraction (`internal/utils/trace_id.go`)
- `GetTraceID()` prioritizes OTel trace ID
- Falls back to Sentry trace ID for backward compatibility
- Used in request logging to correlate logs with traces

## Headers Propagated to Downstream Services

### W3C Trace Context (Standard)
```
traceparent: 00-<trace-id>-<span-id>-<flags>
tracestate: <vendor-specific-data>
```

### Sentry Headers (Backward Compatibility)
```
sentry-trace: <trace-id>-<span-id>-<sampled>
baggage: sentry-trace_id=<trace-id>,sentry-environment=<env>,...
```

## Configuration

The integration uses existing Sentry configuration:

```yaml
monitoring:
  sentry:
    enabled: true
    dsn: "https://..."
    environment: "production"
    sampleRate: 0.1  # Used by both OTel and Sentry
```

- **sampleRate**: Controls trace sampling for both OTel and Sentry
  - `1.0` = sample all traces
  - `0.1` = sample 10% of traces
  - `0.0` = sample no traces

## Benefits

1. **Standards-Based**: Uses W3C Trace Context standard
2. **Vendor-Neutral**: Can add other tracing backends (Jaeger, Tempo) without changing code
3. **Backward Compatible**: Still sends Sentry headers for existing downstream services
4. **Unified Trace IDs**: Single trace ID shared between OTel and Sentry
5. **Future-Proof**: Easy to migrate downstream services to OTel

## Verification

### Check Trace ID in Logs
```bash
# Start the gateway and make a request
curl http://localhost:8080/health

# Check logs for traceID field
# The trace ID should be a 32-character hex string (OTel format)
```

### Check Headers on Outgoing Requests
Downstream services should receive:
```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
sentry-trace: 4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1
baggage: sentry-trace_id=4bf92f3577b34da6a3ce929d0e0e4736,...
```

### Verify in Sentry UI
1. Go to Sentry Performance/Traces
2. Find a trace
3. The trace ID should match the one in your logs
4. Spans should be properly connected

## Migration Path for Downstream Services

1. **Phase 1** (Current): Downstream services read `sentry-trace` header
2. **Phase 2**: Update downstream services to read `traceparent` header (W3C standard)
3. **Phase 3**: Eventually remove Sentry header injection if all services migrated

## Adding Other Tracing Backends

To export traces to Jaeger, Tempo, or other backends:

```go
// In cmd/gateway/otel.go, add exporter to trace provider:
import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

exporter, _ := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint("jaeger:4317"))
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),  // Add this line
    sdktrace.WithSampler(sampler),
    sdktrace.WithResource(res),
    sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
)
```

## Troubleshooting

### Trace IDs not appearing in logs
- Check that Sentry is enabled in config
- Verify OTel middleware is registered before Sentry middleware

### Downstream services not receiving headers
- Check `sentryHeaderInjector` is in the middleware chain
- Verify it's called before the proxy middleware

### Different trace IDs in Sentry vs logs
- Ensure OTel middleware comes before Sentry middleware
- Check that `utils.GetTraceID()` is extracting from OTel context

