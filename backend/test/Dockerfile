# test-service Dockerfile
# 构建：docker build -t auction/test-service:latest backend/test
# 运行：通过 docker-compose（profile=test）启动

# ---------- 构建阶段 ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/test-service .

# ---------- 运行阶段 ----------
FROM alpine:3.19
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /out/test-service /app/test-service

# 主端口 :18090；WS :18092；Mock Partner :18091
EXPOSE 18090 18091 18092

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -q -O /dev/null http://localhost:18090/health || exit 1

ENTRYPOINT ["/app/test-service"]
