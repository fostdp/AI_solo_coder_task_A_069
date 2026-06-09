FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo \
    -o /dccooling-server ./cmd/

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/config.json /app/config.json
COPY --from=builder /dccooling-server /app/dccooling-server
COPY frontend/dist/ /app/frontend/dist/

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["./dccooling-server"]
