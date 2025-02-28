FROM golang:1.24.0 AS build

WORKDIR /tmp/kubedownscaler

COPY ../../go.mod /tmp/kubedownscaler/go.mod

RUN go mod download

COPY ../.. /tmp/kubedownscaler

RUN CGO_ENABLED=0 go build -o bin/admissioncontroller ./cmd/admissioncontroller

FROM scratch

COPY --from=build /tmp/kubedownscaler/bin/admissioncontroller /app/admissioncontroller

ENTRYPOINT ["/app/admissioncontroller"]
