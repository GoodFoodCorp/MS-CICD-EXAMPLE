FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/main.go

FROM gcr.io/distroless/static-debian11

WORKDIR /

COPY --from=builder /app/auth-service /auth-service
COPY --from=builder /app/.env . 
COPY --from=builder /app/docs /docs

USER nonroot:nonroot

EXPOSE 8081

ENTRYPOINT ["/auth-service"]