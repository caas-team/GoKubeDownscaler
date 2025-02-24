# syntax=docker/dockerfile:experimental
# ---
FROM golang:1.24.0 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /tmp/kubedownscaler
COPY ../.. /tmp/kubedownscaler

# Build admission-webhook
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=private \
  go build -o bin/admission-webhook ./cmd/admissioncontroller

# ---
FROM scratch AS run

COPY --from=build /tmp/kubedownscaler/bin/admission-webhook /usr/local/bin/

CMD ["/usr/local/bin/admission-webhook"]
