FROM golang:1.22.5 AS build

WORKDIR /tmp/kubedownscaler

COPY ./go.mod /tmp/kubedownscaler/go.mod

RUN go mod download

COPY . /tmp/kubedownscaler

RUN go build -o bin/gokubedownscaler ./cmd/kubedownscaler

FROM scratch

COPY --from=build /tmp/kubedownscaler/bin/gokubedownscaler /app/backend

CMD ["/app/gokubedownscaler"]
