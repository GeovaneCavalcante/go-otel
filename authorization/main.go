package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go-otel/internal/opentelemetry"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	serviceName    = "authorization-api"
	serviceVersion = "1.1.0"
	port           = ":8081"
)

type AuthRequest struct {
	UserID string `json:"userId"`
}

func newResource() (*resource.Resource, error) {
	ctx := context.Background()
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
}

func main() {

	otelResource, _ := newResource()
	tp, err := opentelemetry.InitTracer(otelResource)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	mp, err := opentelemetry.InitMeter(otelResource)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}()

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(authHandler), "handler-authorization")

	http.Handle("/authorization", otelHandler)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meter := otel.Meter(serviceName)
	tr := otel.Tracer(serviceName)

	initAuthCounter, _ := meter.Int64Counter("init-auth", metric.WithUnit("0"))
	errorAuthCounter, _ := meter.Int64Counter("error-auth", metric.WithUnit("0"))

	initAuthCounter.Add(ctx, 1)
	errorAuthCounter.Add(ctx, 0)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var authReq AuthRequest
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&authReq)
	if err != nil {
		errorAuthCounter.Add(ctx, 1)
		w.WriteHeader(http.StatusBadRequest)
		errorMessage := map[string]string{
			"error": "Missing or invalid fields",
		}
		json.NewEncoder(w).Encode(errorMessage)
		return

	}

	err = validateUser(authReq.UserID)
	ctx, span := tr.Start(ctx, "user-validation")
	span.SetAttributes(attribute.String("userId", authReq.UserID))
	if err != nil {
		errorAuthCounter.Add(ctx, 1)
		span.SetStatus(codes.Error, "unauthorized user")
		span.End()
		w.WriteHeader(http.StatusUnauthorized)
		errorMessage := map[string]string{
			"error": err.Error(),
		}
		json.NewEncoder(w).Encode(errorMessage)
		return
	}
	span.SetStatus(codes.Ok, "user authorized successfully")
	span.End()

	w.WriteHeader(http.StatusOK)
	message := map[string]string{
		"token": generateHash(authReq.UserID),
	}

	json.NewEncoder(w).Encode(message)
}

func validateUser(userID string) error {
	time.Sleep(1 * time.Second)
	if userID == "" || userID != "123" {
		return errors.New("invalid user")
	}
	return nil
}

func generateHash(data string) string {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
