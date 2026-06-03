FROM docker.m.daocloud.io/library/golang:1.25-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

COPY product/go.mod product/go.sum ./product/
COPY shared/llm/go.mod ./shared/llm/
WORKDIR /src/product
RUN go mod download

WORKDIR /src
COPY product ./product
COPY shared/llm ./shared/llm
WORKDIR /src/product
RUN go build -o /out/product-service .

FROM docker.m.daocloud.io/library/alpine:3.19
WORKDIR /app
COPY --from=builder /out/product-service /app/product-service

EXPOSE 8081

ENTRYPOINT ["/app/product-service"]
