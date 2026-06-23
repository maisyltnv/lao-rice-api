# Build
FROM golang:1.23-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /server ./server
COPY --from=builder /app/images ./images

RUN mkdir -p /app/uploads/payment-receipts /app/uploads/product-images /app/uploads/banner-images && chmod -R 777 /app/uploads

ENV PORT=8080
ENV UPLOAD_DIR=/app/uploads
EXPOSE 8080

ENTRYPOINT ["/app/server"]
