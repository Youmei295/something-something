# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache module downloads separately from source for fast rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build every entrypoint into one image. The compose files pick the binary via
# `command`, so the same image backs the monolith and the microservice layout.
RUN set -eux; \
    for svc in api worker migrate identity submission conversation attachment gateway; do \
        CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o "/out/$svc" "./cmd/$svc"; \
    done

# ---- runtime ----
# Distroless static includes CA certificates and a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/ /usr/local/bin/

USER nonroot:nonroot
EXPOSE 8080

# CMD (not ENTRYPOINT) so compose can override the command for the worker and
# migrate services without appending arguments to the api binary.
CMD ["/usr/local/bin/api"]
