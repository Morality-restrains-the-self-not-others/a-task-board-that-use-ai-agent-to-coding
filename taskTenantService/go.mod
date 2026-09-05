module taskTenantService

go 1.24

require (
	daydaymoneymeta v0.0.0
	confload v0.0.0
	dbload v0.0.0
	gatewayauth v0.0.0
	gatewaycors v0.0.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/segmentio/kafka-go v0.4.47
	mysqlmeta v0.0.0
	snowflake v0.0.0
	tracelog v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.20.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

require (
	authz v0.0.0
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace confload => ../shareLib/confload

replace gatewayauth => ../shareLib/gatewayauth

replace tracelog => ../shareLib/tracelog

replace gatewaycors => ../shareLib/gatewaycors

replace daydaymoneymeta => ../shareLib/daydaymoneymeta

replace snowflake => ../shareLib/snowflake

replace mysqlmeta => ../shareLib/mysqlmeta

replace dbload => ../db/load

replace authz => ../shareLib/authz
