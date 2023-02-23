module github.com/chrislgardner/spellapi

go 1.16

require (
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0
	go.mongodb.org/mongo-driver v1.7.3
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.25.0
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.0.1
	go.opentelemetry.io/otel/sdk v1.0.1
	golang.org/x/text v0.3.8 // indirect
	google.golang.org/grpc v1.41.0
	gopkg.in/launchdarkly/go-sdk-common.v2 v2.2.2
	gopkg.in/launchdarkly/go-server-sdk.v5 v5.3.0
)
