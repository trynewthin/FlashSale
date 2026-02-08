// tracing 包包含相关应用代码。
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TraceConfig 描述链路追踪初始化参数。
type TraceConfig struct {
	Endpoint    string
	Insecure    bool
	ServiceName string
}

// Init 初始化全局 TracerProvider，并返回关闭函数。
func Init(cfg TraceConfig) (func(context.Context) error, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "flashsale-base"
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("", attribute.String("service.name", serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("build resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return provider.Shutdown, nil
}
