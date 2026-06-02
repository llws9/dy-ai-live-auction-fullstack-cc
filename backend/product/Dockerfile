FROM docker.m.daocloud.io/library/golang:1.25-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/product-service .

FROM docker.m.daocloud.io/library/alpine:3.19
WORKDIR /app
COPY --from=builder /out/product-service /app/product-service

EXPOSE 8081

ENTRYPOINT ["/app/product-service"]
