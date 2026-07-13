module testdata

go 1.26.4

require (
	github.com/moov-io/base v0.0.0
	github.com/moovfinancial/errors v0.0.0
	github.com/moovfinancial/go-libs v0.0.0
	go.opentelemetry.io/otel/trace v0.0.0
	cloud.google.com/go/spanner v0.0.0
	google.golang.org/grpc/codes v0.0.0
)

replace github.com/moov-io/base => ./moovio_base
replace github.com/moovfinancial/errors => ./errors
replace github.com/moovfinancial/go-libs => ./go_libs
replace go.opentelemetry.io/otel/trace => ./otel_trace
replace cloud.google.com/go/spanner => ./spanner
replace google.golang.org/grpc/codes => ./codes
