FROM docker.m.daocloud.io/library/golang:1.25-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/auction-service .

FROM docker.m.daocloud.io/library/alpine:3.19
WORKDIR /app
COPY --from=builder /out/auction-service /app/auction-service

EXPOSE 8082 8083

ENTRYPOINT ["/app/auction-service"]
