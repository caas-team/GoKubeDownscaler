FROM --platform=$BUILDPLATFORM golang:1.25.8 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /tmp/kubedownscaler

COPY ../../go.mod /tmp/kubedownscaler/go.mod

RUN go mod download

COPY ../.. /tmp/kubedownscaler

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/gokubedownscaler ./cmd/kubedownscaler

FROM scratch

COPY --from=build /tmp/kubedownscaler/bin/gokubedownscaler /app/gokubedownscaler

ENTRYPOINT ["/app/gokubedownscaler"]
