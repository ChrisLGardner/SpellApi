package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/chrislgardner/spellapi/db"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var (
	dbUrl string
)

func main() {

	ctx, tp := initHoneycomb()
	// Handle this error in a sensible manner where possible
	defer func() { _ = tp.Shutdown(ctx) }()

	dbUrl = os.Getenv("COSMOSDB_URI")
	db, err := db.ConnectDb(dbUrl)
	if err != nil {
		panic(err)
	}

	var spellService SpellService
	if ldApiKey := os.Getenv("LAUNCHDARKLY_KEY"); ldApiKey != "" {
		ldclient, err := NewLaunchDarklyClient(ldApiKey, 5)
		if err != nil {
			panic(err)
		}

		spellService = SpellService{
			store: db,
			flags: ldclient,
		}
	} else {
		spellService = SpellService{
			store: db,
		}
	}

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("SpellApi"))
	// Routes consist of a path and a handler function.
	r.HandleFunc("/spells/{name}", spellService.GetSpellHandler).Methods("GET")
	r.HandleFunc("/spells/{name}", spellService.DeleteSpellHandler).Methods("DELETE")
	r.HandleFunc("/spells", spellService.PostSpellHandler).Methods("POST")
	r.HandleFunc("/spells", spellService.GetAllSpellHandler).Methods("GET")
	r.HandleFunc("/spellmetadata/{name}", spellService.GetSpellMetadataHandler).Methods("GET")
	r.HandleFunc("/spellmetadata", spellService.GetAllSpellMetadataHandler).Methods("GET")

	// Bind to a port and pass our router in
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func initHoneycomb() (context.Context, *sdktrace.TracerProvider) {
	ctx := context.Background()

	// Create an OTLP exporter, passing in Honeycomb credentials as environment variables.
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    os.Getenv("HONEYCOMB_KEY"),
			"x-honeycomb-dataset": os.Getenv("HONEYCOMB_DATASET"),
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	)

	if err != nil {
		fmt.Printf("failed to initialize exporter: %v", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("Encantus"),
		),
	)
	if err != nil {
		fmt.Printf("failed to initialize respource: %v", err)
	}
	// Create a new tracer provider with a batch span processor and the otlp exporter.
	// Add a resource attribute service.name that identifies the service in the Honeycomb UI.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	// Set the Tracer Provider and the W3C Trace Context propagator as globals
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)

	return ctx, tp
}
