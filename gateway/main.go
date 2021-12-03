package main

import (
	"fmt"
	pb "github.com/pmorelli92/open-telemetry-go/proto"
	"github.com/pmorelli92/open-telemetry-go/utils"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
)

func main() {
	jaegerAddress := utils.EnvString("JAEGER_ADDRESS", "localhost")
	jaegerPort := utils.EnvString("JAEGER_PORT", "6831")
	checkoutAddress := utils.EnvString("CHECKOUT_SERVICE_ADDRESS", "localhost:8080")
	httpAddress := utils.EnvString("HTTP_ADDRESS", ":8081")

	err := utils.SetGlobalTracer("gateway", jaegerAddress, jaegerPort)
	if err != nil {
		log.Fatalf("failed to create tracer: %v", err)
	}

	conn, err := grpc.Dial(
		checkoutAddress,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewCheckoutClient(conn)

	// HTTP config
	router := http.NewServeMux()
	router.HandleFunc("/api/checkout", checkoutHandler(c))
	fmt.Println("HTTP server listening at ", httpAddress)
	log.Fatal(http.ListenAndServe(httpAddress, router))
}

func checkoutHandler(c pb.CheckoutClient) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow only POST
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Create a tracer span
		tr := otel.Tracer("http")
		ctx, span := tr.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.RequestURI))
		defer span.End()

		// Make the GRPC call to checkout-service
		_, err := c.DoCheckout(ctx, &pb.CheckoutRequest{
			ItemsID: []int32{1, 2, 3, 4},
		})

		// Check for errors
		rStatus := status.Convert(err)
		if rStatus != nil {
			span.SetStatus(codes.Error, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success
		w.WriteHeader(http.StatusAccepted)
	}
}
