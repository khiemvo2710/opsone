# OpsOne API — GreenNode AgentBase Runtime (port 8080, GET /health)
FROM golang:1.21.13-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

FROM alpine:3.20.3
RUN apk add --no-cache ca-certificates tzdata
ENV APP_TIMEZONE=Asia/Ho_Chi_Minh
WORKDIR /app
COPY --from=build /api .
EXPOSE 8080
ENV API_ADDR=:8080
CMD ["/app/api"]
