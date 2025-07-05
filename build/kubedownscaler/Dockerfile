FROM golang:1.24.4 AS build

WORKDIR /tmp/kubedownscaler

COPY ../../go.mod /tmp/kubedownscaler/go.mod

RUN go mod download

COPY ../.. /tmp/kubedownscaler

RUN CGO_ENABLED=0 go build -o bin/gokubedownscaler ./cmd/kubedownscaler

FROM scratch

COPY --from=build /tmp/kubedownscaler/bin/gokubedownscaler /app/gokubedownscaler

ENTRYPOINT ["/app/gokubedownscaler"]
