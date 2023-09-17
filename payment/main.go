package main

import (
	"bytes"
	"context"
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
	serviceName      = "payment-api"
	serviceVersion   = "1.1.0"
	port             = ":8080"
	authorizationUrl = "http://localhost:8081/authorization"
)

const (
	PaymentFailedNotProcessed = "payment failed at broker and not processed"
	PaymentRequestError       = "payment error during request to broker"
	ProductInvalidAmount      = "amount must be positive and greater than zero"
	ProductInvalidID          = "productId cannot be an empty string"
)

type PaymentRequest struct {
	ProductID string  `json:"productId"`
	Amount    float64 `json:"amount"`
	UserID    string  `json:"userId"`
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

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(paymentHandler), "handler-payment")

	http.Handle("/payment", otelHandler)
	err = http.ListenAndServe(port, nil)
	if err != nil {

		log.Fatal(err)
	}
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	meter := otel.Meter(serviceName)
	tr := otel.Tracer(serviceName)

	initPaymentCounter, _ := meter.Int64Counter("init-payment", metric.WithUnit("0"))
	errorPaymentCounter, _ := meter.Int64Counter("error-payment", metric.WithUnit("0"))

	initPaymentCounter.Add(ctx, 1)
	errorPaymentCounter.Add(ctx, 0)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var paymentReq PaymentRequest
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&paymentReq)
	if err != nil {
		errorPaymentCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("error", err.Error())))
		w.WriteHeader(http.StatusBadRequest)
		errorMessage := map[string]string{
			"error": "Missing or invalid fields",
		}
		json.NewEncoder(w).Encode(errorMessage)
		return

	}

	ctx, span := tr.Start(ctx, "product-validation")
	err = productValidation(paymentReq.ProductID, paymentReq.Amount)
	if err != nil {
		errorPaymentCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("error", err.Error())))
		span.SetStatus(codes.Error, "error when validating product")
		span.End()
		w.WriteHeader(http.StatusUnauthorized)
		errorMessage := map[string]string{
			"error": err.Error(),
		}
		json.NewEncoder(w).Encode(errorMessage)
		return
	}
	span.SetStatus(codes.Ok, "product successfully validated")
	span.End()

	err = authorizePayment(ctx, paymentReq.UserID)
	if err != nil {
		errorPaymentCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("error", err.Error())))
		w.WriteHeader(http.StatusUnauthorized)
		errorMessage := map[string]string{
			"error": "Could not authorize payment",
		}
		json.NewEncoder(w).Encode(errorMessage)
		return
	}

	tpvCounter, _ := meter.Float64Counter("tpv", metric.WithUnit("0"))
	tpvCounter.Add(ctx, paymentReq.Amount, metric.WithAttributes(attribute.String("productId", paymentReq.ProductID)))

	w.WriteHeader(http.StatusOK)
	message := map[string]string{
		"message": "payment made successfully",
	}

	json.NewEncoder(w).Encode(message)

}
func productValidation(productID string, amount float64) error {

	time.Sleep(2 * time.Second)
	if productID == "" {
		return errors.New(ProductInvalidID)
	}

	if amount <= 0 {
		return errors.New(ProductInvalidAmount)
	}

	return nil
}

func authorizePayment(ctx context.Context, userID string) error {

	dataMap := map[string]interface{}{
		"userId": userID,
	}

	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	req, err := http.NewRequestWithContext(ctx, "POST", authorizationUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("error authorizing user")
	}

	defer resp.Body.Close()

	return nil
}
