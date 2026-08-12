# Continuous CRUD generator for the inventory schema (Go port of datagen/gen.py).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/datagen ./cmd/datagen

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 app
USER app
COPY --from=build /out/datagen /usr/local/bin/datagen
CMD ["datagen"]
