# gateway-service Dockerfile
# 测试平台前端的 /api 和 /ws 必须通过 gateway 转发到 test-service。

FROM docker.m.daocloud.io/library/golang:1.25-alpine AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/gateway-service .

FROM docker.m.daocloud.io/library/alpine:3.19
WORKDIR /app
COPY --from=builder /out/gateway-service /app/gateway-service

EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gateway-service"]
