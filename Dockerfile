FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/joy-operator ./cmd/operator

FROM alpine

COPY --from=builder /bin/joy-operator /usr/local/bin/joy-operator

RUN openssh-client ca-certificates && update-ca-certificates 2>/dev/null || true

SHELL ["/bin/sh", "-o", "pipefail", "-c"]

RUN \
  apk add perl-utils && \
  wget https://get.helm.sh/helm-v3.18.5-linux-amd64.tar.gz -q && \
  echo "9879bf9c471cdecbbee5ee17cf1de1849b0ffd12871ea01f17ede6861d7134f5  helm-v3.18.5-linux-amd64.tar.gz" | shasum -a256 --check - && \
  tar -xzf helm-v3.18.5-linux-amd64.tar.gz && \
  mv linux-amd64/helm /usr/local/bin

ENTRYPOINT ["/usr/local/bin/joy-operator"]
