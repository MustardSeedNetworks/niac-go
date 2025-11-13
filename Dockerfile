FROM golang:1.24 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/niac ./cmd/niac

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /out/niac /usr/local/bin/niac
COPY examples ./examples
ENTRYPOINT ["niac"]
