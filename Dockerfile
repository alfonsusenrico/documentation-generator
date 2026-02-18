# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/docgen ./cmd/docgen

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    gh \
    pandoc \
    texlive-xetex \
    texlive-latex-base \
    texlive-latex-recommended \
    texlive-fonts-recommended \
    lmodern \
    fonts-dejavu \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 -s /bin/bash docgen

WORKDIR /app
COPY --from=builder /out/docgen /app/docgen
COPY public /app/public
COPY templates /app/templates
COPY .env.example /app/.env.example

USER docgen

EXPOSE 8001
ENTRYPOINT ["/app/docgen"]
