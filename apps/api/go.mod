module github.com/bengutow1/csc4330_project5/apps/api

go 1.26.4

require (
	connectrpc.com/connect v1.21.0
	github.com/bengutow1/csc4330_project5/packages/go/apigen v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/handlers v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/logging v0.0.0
	github.com/bengutow1/csc4330_project5/packages/go/repository v0.0.0
	github.com/go-chi/chi/v5 v5.3.2
	github.com/jackc/pgx/v5 v5.11.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.15 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/leodido/go-urn v1.5.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/bengutow1/csc4330_project5/packages/go/apigen => ../../packages/go/apigen
	github.com/bengutow1/csc4330_project5/packages/go/handlers => ../../packages/go/handlers
	github.com/bengutow1/csc4330_project5/packages/go/logging => ../../packages/go/logging
	github.com/bengutow1/csc4330_project5/packages/go/repository => ../../packages/go/repository
)
