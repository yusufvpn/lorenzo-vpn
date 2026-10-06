# Stage 1: Build the Lorenzo VPN Server in pure Go
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o lorenzo-server .

# Stage 2: Minimal Production Image
FROM alpine:latest

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/lorenzo-server /app/lorenzo-server
COPY server/web /app/web

ENV PORT=8080
ENV LORENZO_SECRET=LorenzoStrictLeaderSecret2026

EXPOSE 8080

CMD ["/app/lorenzo-server"]
