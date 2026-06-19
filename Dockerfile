FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /solar ./cmd/solar

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /solar /solar
COPY configs/server.toml /configs/server.toml

EXPOSE 25565
USER nonroot:nonroot

ENTRYPOINT ["/solar", "start", "--config", "/configs/server.toml"]
