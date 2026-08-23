# syntax=docker/dockerfile:1
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS build

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=readonly

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal

RUN go build \
    -trimpath \
    -ldflags="-s -w -buildid= -X main.generatorVersion=0.1.0-dev" \
    -o /out/breachsafe-report \
    ./cmd/evidence-report

FROM scratch

LABEL org.opencontainers.image.title="BreachSAFE PDF report compiler" \
      org.opencontainers.image.description="Pure-Go BreachSAFE PDF report compiler" \
      org.opencontainers.image.version="0.1.0-dev" \
      org.opencontainers.image.licenses="PolyForm-Noncommercial-1.0.0"

WORKDIR /work
COPY --from=build --chown=65532:65532 /out/breachsafe-report /usr/local/bin/breachsafe-report

USER 65532:65532
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/breachsafe-report"]
