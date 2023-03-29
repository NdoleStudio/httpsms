module github.com/NdoleStudio/httpsms

go 1.18

require (
	cloud.google.com/go/cloudtasks v1.10.0
	firebase.google.com/go v3.13.0+incompatible
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.36.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.12.0
	github.com/NdoleStudio/go-otelroundtripper v0.0.7
	github.com/NdoleStudio/lemonsqueezy-go v0.0.8
	github.com/carlmjohnson/requests v0.23.2
	github.com/cheggaaa/pb/v3 v3.0.8
	github.com/cloudevents/sdk-go/v2 v2.13.0
	github.com/cockroachdb/cockroach-go/v2 v2.3.3
	github.com/davecgh/go-spew v1.1.1
	github.com/gofiber/fiber/v2 v2.42.0
	github.com/gofiber/swagger v0.1.9
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-retryablehttp v0.7.2
	github.com/hirosassa/zerodriver v0.1.4
	github.com/jinzhu/now v1.1.5
	github.com/joho/godotenv v1.5.1
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/lib/pq v1.10.7
	github.com/matcornic/hermes/v2 v2.1.0
	github.com/nyaruka/phonenumbers v1.1.6
	github.com/palantir/stacktrace v0.0.0-20161112013806-78658fd2d177
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.0.2
	github.com/rs/zerolog v1.29.0
	github.com/sendgrid/sendgrid-go v3.12.0+incompatible
	github.com/swaggo/swag v1.8.10
	github.com/thedevsaddam/govalidator v1.9.10
	github.com/uptrace/uptrace-go v1.13.0
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/metric v0.37.0
	go.opentelemetry.io/otel/sdk v1.14.0
	go.opentelemetry.io/otel/sdk/metric v0.37.0
	go.opentelemetry.io/otel/trace v1.14.0
	google.golang.org/api v0.114.0
	google.golang.org/genproto v0.0.0-20230320173215-1fe4d14fc725
	google.golang.org/protobuf v1.30.0
	gorm.io/datatypes v1.1.1
	gorm.io/driver/postgres v1.5.0
	gorm.io/gorm v1.24.7-0.20230306060331-85eaf9eeda11
)

require (
	cloud.google.com/go v0.110.0 // indirect
	cloud.google.com/go/compute v1.18.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/firestore v1.9.0 // indirect
	cloud.google.com/go/iam v0.13.0 // indirect
	cloud.google.com/go/longrunning v0.4.1 // indirect
	cloud.google.com/go/monitoring v1.12.0 // indirect
	cloud.google.com/go/storage v1.30.0 // indirect
	cloud.google.com/go/trace v1.9.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.36.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.8 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.8.0 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.14 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.3.1 // indirect
	github.com/jaytaylor/html2text v0.0.0-20211105163654-bc68cce691ba // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/savsgio/dictpool v0.0.0-20221023140959-7bf2e61cea94 // indirect
	github.com/savsgio/gotils v0.0.0-20230208104028-c358bd845dee // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/swaggo/files v1.0.0 // indirect
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.45.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/vanng822/css v1.0.1 // indirect
	github.com/vanng822/go-premailer v1.20.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.14.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/oauth2 v0.6.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/grpc v1.53.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.4.7 // indirect
)
