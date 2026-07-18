# syntax=docker/dockerfile:1

# Build stage: generate templ code, compile Tailwind CSS and the Go binary.
FROM golang:1.26-bookworm AS build
WORKDIR /src

ARG TEMPL_VERSION=v0.3.1020
ARG TAILWIND_VERSION=v3.4.17
RUN go install github.com/a-h/templ/cmd/templ@${TEMPL_VERSION} \
    && curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-x64" -o /usr/local/bin/tailwindcss \
    && chmod +x /usr/local/bin/tailwindcss

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate \
    && tailwindcss -i web/static/css/input.css -o web/static/css/main.css --minify \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/signal ./cmd/server

# Runtime stage: static binary only, runs as non-root.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /etc/signal
COPY --from=build /out/signal /usr/local/bin/signal
EXPOSE 8080
ENTRYPOINT ["signal"]
CMD ["-config", "/etc/signal/config.yaml"]
