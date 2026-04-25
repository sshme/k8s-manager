FROM golang:alpine AS builder

WORKDIR /app

COPY proto ./proto

COPY market/go.mod market/go.sum ./market/
RUN cd market && go mod download

COPY market ./market
    
# Build the application from cmd/market directory
# -ldflags="-s -w" removes debug symbols to reduce binary size
RUN cd market && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o market-service ./cmd/market

FROM alpine:latest
    
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/market/market-service .
COPY --from=builder /app/market/migrations ./migrations

EXPOSE 50051

CMD ["./market-service"]
