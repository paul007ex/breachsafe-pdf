# syntax=docker/dockerfile:1
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS build

ARG REPORT_VERSION=0.1.1

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=readonly

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal

RUN go build \
    -trimpath \
    -ldflags="-s -w -buildid= -X main.version=${REPORT_VERSION}" \
    -o /out/breachsafe-pdf \
    ./cmd/breachsafe-pdf

RUN go build \
    -trimpath \
    -ldflags="-s -w -buildid= -X main.generatorVersion=${REPORT_VERSION}" \
    -o /out/breachsafe-report \
    ./cmd/evidence-report

FROM scratch

ARG REPORT_VERSION=0.1.1

LABEL org.opencontainers.image.title="BreachSAFE PDF report compiler" \
      org.opencontainers.image.description="Pure-Go BreachSAFE PDF report compiler" \
      org.opencontainers.image.version="${REPORT_VERSION}" \
      org.opencontainers.image.licenses="PolyForm-Noncommercial-1.0.0"

WORKDIR /work
COPY --from=build --chown=65532:65532 /out/breachsafe-report /usr/local/bin/breachsafe-report
COPY --from=build --chown=65532:65532 /out/breachsafe-pdf /usr/local/bin/breachsafe-pdf

USER 65532:65532
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/breachsafe-pdf"]
