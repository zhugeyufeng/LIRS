FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY apps/server/go.mod apps/server/go.sum* ./
RUN go mod download
COPY apps/server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/lirs-api ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /out/lirs-api /app/lirs-api
EXPOSE 8080
CMD ["/app/lirs-api"]

