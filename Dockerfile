# syntax=docker/dockerfile:1
#
# Stage graph:  web ─┐
#                    gobase ─┬─ build ──────────┐
#                            ├─ lint            │
#               native ──────┴─ test   native ──┴─ runtime
#
# Only web contains Node; only gobase/build/lint/test contain Go. test and runtime share
# the native stage, so the tests run against exactly the ExifTool/libvips/ffmpeg the NAS
# container ships.

FROM node:24-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-trixie AS gobase
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY web/*.go ./web/
COPY testdata ./testdata
COPY docs/concepts/photocurator-metadaten-strategie.md ./docs/concepts/photocurator-metadaten-strategie.md
COPY --from=web /src/web/dist ./web/dist

FROM gobase AS build
ARG VERSION=0.1.0-dev
RUN go build -trimpath -ldflags "-s -w -X photocurator/internal/version.Version=${VERSION}" \
      -o /out/photocurator ./cmd/photocurator

FROM gobase AS lint
RUN unformatted="$(gofmt -l .)"; \
    if [ -n "$unformatted" ]; then echo "gofmt needed for:"; echo "$unformatted"; exit 1; fi; \
    go vet ./...

FROM debian:trixie-slim AS native
# libheif in trixie decodes HEIC only through a codec plugin; libde265 is the decode-only one.
# --no-install-recommends would otherwise drop it silently. libheif-plugin-x265 adds the HEVC
# encoder that `photocurator probe generate` needs (vips heifsave) to create HEIC probe files.
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      libimage-exiftool-perl libvips-tools libheif-plugin-libde265 libheif-plugin-x265 ffmpeg tini ca-certificates \
 && rm -rf /var/lib/apt/lists/*

FROM native AS test
COPY --from=gobase /usr/local/go /usr/local/go
COPY --from=gobase /go /go
# CGO stays off: the native stage has no C toolchain, and the shipped binary is static too.
ENV PATH=/usr/local/go/bin:/go/bin:$PATH GOPATH=/go GOCACHE=/tmp/gocache \
    CGO_ENABLED=0 GOTOOLCHAIN=local
WORKDIR /src
COPY --from=gobase /src /src
RUN go test ./...

FROM native AS runtime
RUN groupadd --system --gid 10001 photocurator \
 && useradd --system --uid 10001 --gid 10001 --no-create-home --home-dir /nonexistent \
      --shell /usr/sbin/nologin photocurator
COPY --from=build /out/photocurator /app/photocurator
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["tini", "--", "/app/photocurator"]
CMD ["serve"]
