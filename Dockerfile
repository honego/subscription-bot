FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/subscription-bot ./cmd/subscription-bot

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
VOLUME ["/app/data"]
COPY --from=builder /out/subscription-bot /app/subscription-bot
ENTRYPOINT ["/app/subscription-bot"]
