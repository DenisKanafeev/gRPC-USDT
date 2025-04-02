package optel

import (
	"testing"

	"go.opentelemetry.io/otel"
)

func TestInitTracer(t *testing.T) {
	otlpEndpoint := "http://localhost:14268/v1/traces"
	serviceName := "test-service"

	tp, err := InitTracer(otlpEndpoint, serviceName)
	if err != nil {
		t.Errorf("InitTracer failed: %v", err)
	}

	if tp == nil {
		t.Errorf("TracerProvider is nil")
	}

	// Проверка того, что провайдер установлен глобально
	if otel.GetTracerProvider() == nil {
		t.Errorf("Global TracerProvider is not set")
	}
}

func TestInitTracer_EmptyServiceName(t *testing.T) {
	otlpEndpoint := "http://localhost:14268/v1/traces"
	serviceName := ""

	tp, err := InitTracer(otlpEndpoint, serviceName)
	if err != nil {
		t.Errorf("InitTracer failed: %v", err)
	}

	if tp == nil {
		t.Errorf("TracerProvider is nil")
	}
}
