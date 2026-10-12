# syntax=docker/dockerfile:1
#
# The api image. Build from the repo root (`moon run api:image`):
#
#   docker build -f apps/api/Dockerfile --build-arg GIT_SHA=$(git rev-parse --short HEAD) .
#
# The whole of packages/go is copied because go.work names every module in it; the linker keeps
# only what apps/api imports, so a new package needs no change here.

FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
WORKDIR /src

# The [m] glob makes go.work.sum optional: go only writes it once the workspace needs a checksum
# that no module's go.sum holds.
COPY go.work go.work.su[m] ./
COPY packages/go ./packages/go
COPY apps/api ./apps/api

WORKDIR /src/apps/api
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags='-s -w' -o /out/api .

# A static binary on a minimal base that runs as nonroot.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
# The commit this image was built from, read by packages/go/logging as APP_VERSION.
ARG GIT_SHA
ENV APP_VERSION=$GIT_SHA
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
