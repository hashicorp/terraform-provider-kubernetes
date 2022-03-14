FROM golang:1.17 AS builder
WORKDIR /src/k8s-manifest-hook
ADD *.go ./
ADD go.* ./
RUN CGO_ENABLED=0 go build

FROM alpine:latest
COPY  --from=builder /src/k8s-manifest-hook/k8s-manifest-hook /k8s-manifest-hook
VOLUME /etc/webhook/certs
ENTRYPOINT /k8s-manifest-hook