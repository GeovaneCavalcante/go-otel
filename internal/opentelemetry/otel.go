package opentelemetry

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/exporters/zipkin"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	zipkinUrl    = "http://localhost:9411/api/v2/spans"
	collectorUrl = "localhost:4317"
)

// Falar sobre a API de trace do Otel, do porque das especificações e como eu posso usar vários Sdks dela
func InitTracer(r *resource.Resource) (*sdktrace.TracerProvider, error) {
	// Falar sobre SDK, o que é e o que faz. E quais SDKs disponíveis, e como faço pra construir meu próprio SDK
	exporter, err := zipkin.New(
		zipkinUrl,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Falar sobre o Sample e estratégias para utilizar-lo
		sdktrace.WithBatcher(exporter),                // Falar sobre os Exporters
		sdktrace.WithResource(r),                      // Falar sobre o Resource p que é e pra que serve
	)
	otel.SetTracerProvider(tp)                                                                                              // Explicar como funciona o trace provider
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})) // Explicar a propagação de contexto entre serviços
	return tp, nil
}

func InitMeter(r *resource.Resource) (*sdkmetric.MeterProvider, error) {
	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, collectorUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}
	metricExporter, _ := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(r),
	)
	otel.SetMeterProvider(mp)
	return mp, nil
}
