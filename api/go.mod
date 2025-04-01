module github.com/NdoleStudio/httpsms

go 1.22.7
toolchain go1.24.1

require (
	cloud.google.com/go/cloudtasks v1.13.3
	firebase.google.com/go v3.13.0+incompatible
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.27.0
	github.com/NdoleStudio/go-otelroundtripper v0.0.11
	github.com/NdoleStudio/lemonsqueezy-go v1.2.4
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/carlmjohnson/requests v0.24.3
	github.com/cloudevents/sdk-go/v2 v2.15.2
	github.com/cockroachdb/cockroach-go/v2 v2.4.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/ristretto v1.0.0
	github.com/dustin/go-humanize v1.0.1
	github.com/gofiber/contrib/otelfiber v1.0.10
	github.com/gofiber/fiber/v2 v2.52.6
	github.com/gofiber/swagger v1.1.1
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/hirosassa/zerodriver v0.1.4
	github.com/jinzhu/now v1.1.5
	github.com/joho/godotenv v1.5.1
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/jszwec/csvutil v1.10.0
	github.com/lib/pq v1.10.9
	github.com/matcornic/hermes/v2 v2.1.0
	github.com/nyaruka/phonenumbers v1.5.0
	github.com/palantir/stacktrace v0.0.0-20161112013806-78658fd2d177
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/extra/redisotel/v9 v9.7.1
	github.com/redis/go-redis/v9 v9.7.3
	github.com/rs/zerolog v1.34.0
	github.com/sendgrid/sendgrid-go v3.16.0+incompatible
	github.com/stretchr/testify v1.10.0
	github.com/swaggo/swag v1.16.4
	github.com/thedevsaddam/govalidator v1.9.10
	github.com/uptrace/uptrace-go v1.34.0
	github.com/xuri/excelize/v2 v2.9.0
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/metric v1.34.0
	go.opentelemetry.io/otel/sdk v1.34.0
	go.opentelemetry.io/otel/sdk/metric v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
	google.golang.org/api v0.223.0
	google.golang.org/protobuf v1.36.5
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.12
	gorm.io/plugin/opentelemetry v0.1.11
)

require (
	cloud.google.com/go v0.118.0 // indirect
	cloud.google.com/go/auth v0.15.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.7 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/firestore v1.17.0 // indirect
	cloud.google.com/go/iam v1.3.1 // indirect
	cloud.google.com/go/longrunning v0.6.4 // indirect
	cloud.google.com/go/monitoring v1.22.1 // indirect
	cloud.google.com/go/storage v1.43.0 // indirect
	cloud.google.com/go/trace v1.11.3 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/PuerkitoBio/goquery v1.9.2 // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jaytaylor/html2text v0.0.0-20230321000545-74c2419ad056 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.7.1 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/swaggo/files/v2 v2.0.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.54.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/vanng822/css v1.0.1 // indirect
	github.com/vanng822/go-premailer v1.21.0 // indirect
	github.com/xuri/efp v0.0.0-20240408161823-9ad904a10d6d // indirect
	github.com/xuri/nfp v0.0.0-20240318013403-ab9948c2c4a7 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib v1.27.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.59.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.59.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.10.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.34.0 // indirect
	go.opentelemetry.io/otel/log v0.10.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.10.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/exp v0.0.0-20240525044651-4c93da0ed11d // indirect
	golang.org/x/net v0.36.0 // indirect
	golang.org/x/oauth2 v0.26.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/time v0.10.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20250106144421-5f5ef82da422 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250219182151-9fdb1cabc7b2 // indirect
	google.golang.org/grpc v1.70.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
