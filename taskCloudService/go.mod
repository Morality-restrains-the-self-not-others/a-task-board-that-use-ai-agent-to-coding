module taskCloudService

go 1.24

require (
	daydaymoneymeta v0.0.0
	confload v0.0.0
	dbload v0.0.0
	gatewayauth v0.0.0
	gatewaycors v0.0.0
	github.com/alibabacloud-go/darabonba-openapi v0.1.16
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.14
	github.com/alibabacloud-go/ecs-20140526/v7 v7.0.0
	github.com/alibabacloud-go/sts-20150401 v1.1.2
	github.com/alibabacloud-go/tea v1.3.13
	github.com/go-sql-driver/mysql v1.8.1
	github.com/redis/go-redis/v9 v9.7.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/tencentyun/cos-go-sdk-v5 v0.7.70
	gopkg.in/yaml.v3 v3.0.1
	mysqlmeta v0.0.0
	snowflake v0.0.0
	tracelog v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55 // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
)

require (
	authz v0.0.0
	clientip v0.0.0
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.0 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.0 // indirect
	github.com/alibabacloud-go/tea-utils v1.4.3 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/aliyun/credentials-go v1.4.5 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	registryhost v0.0.0
	stopreason v0.0.0
)

replace confload => ../shareLib/confload

replace gatewayauth => ../shareLib/gatewayauth

replace tracelog => ../shareLib/tracelog

replace github.com/alibabacloud-go/ecs-20140526/v7 => ../sdk/ecs-20140526

replace gatewaycors => ../shareLib/gatewaycors

replace daydaymoneymeta => ../shareLib/daydaymoneymeta

replace snowflake => ../shareLib/snowflake

replace mysqlmeta => ../shareLib/mysqlmeta

replace dbload => ../db/load

replace authz => ../shareLib/authz

replace registryhost => ../shareLib/registryhost

replace stopreason => ../shareLib/stopreason

replace clientip => ../shareLib/clientip
