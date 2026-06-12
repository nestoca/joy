FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/joy-operator ./cmd/operator

FROM alpine

COPY --from=builder /bin/joy-operator /usr/local/bin/joy-operator

ENTRYPOINT ["/usr/local/bin/joy-operator"]
