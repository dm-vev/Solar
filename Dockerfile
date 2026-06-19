FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /solar ./cmd/solar

FROM gcr.io/distroless/static-debian12:nonroot

# Run from the nonroot user's home directory so the default data_dir "data"
# resolves to a writable path (/home/nonroot/data).
WORKDIR /home/nonroot
COPY --from=builder /solar /solar
COPY --from=builder /src/configs/server.toml /configs/server.toml

EXPOSE 25565
USER nonroot:nonroot

ENTRYPOINT ["/solar", "start", "--config", "/configs/server.toml"]
