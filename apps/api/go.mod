module github.com/bengutow1/csc4330_project5/apps/api

go 1.26.4

require (
	connectrpc.com/connect v1.21.0
	github.com/alexedwards/scs/pgxstore v0.0.0-20251002162104-209de6e426de
	github.com/alexedwards/scs/v2 v2.9.0
	github.com/bengutow1/csc4330_project5/packages/go/apigen v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/auth v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/handlers v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/logging v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/places v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/repository v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/vision v0.0.0
	github.com/go-chi/chi/v5 v5.3.2
	github.com/jackc/pgx/v5 v5.11.0
)

require (
	github.com/anthropics/anthropic-sdk-go v1.84.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/bengutow1/csc4330_project5/packages/go/points v0.0.0 // indirect
	github.com/bengutow1/csc4330_project5/packages/go/recipes v0.0.0 // indirect
	github.com/bengutow1/csc4330_project5/packages/go/scoring v0.0.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.15 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.5 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/leodido/go-urn v1.5.0 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/standard-webhooks/standard-webhooks/libraries v0.0.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/bengutow1/csc4330_project5/packages/go/apigen => ../../packages/go/apigen
	github.com/bengutow1/csc4330_project5/packages/go/auth => ../../packages/go/auth
	github.com/bengutow1/csc4330_project5/packages/go/handlers => ../../packages/go/handlers
	github.com/bengutow1/csc4330_project5/packages/go/logging => ../../packages/go/logging
	github.com/bengutow1/csc4330_project5/packages/go/places => ../../packages/go/places
	github.com/bengutow1/csc4330_project5/packages/go/points => ../../packages/go/points
	github.com/bengutow1/csc4330_project5/packages/go/recipes => ../../packages/go/recipes
	github.com/bengutow1/csc4330_project5/packages/go/repository => ../../packages/go/repository
	github.com/bengutow1/csc4330_project5/packages/go/scoring => ../../packages/go/scoring
	github.com/bengutow1/csc4330_project5/packages/go/vision => ../../packages/go/vision
)
