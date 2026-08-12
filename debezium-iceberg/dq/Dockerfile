# Data-quality runner + integration tests (Go port of dq/).
#
# The image carries two binaries:
#   dq              - `dq run` (DQ loop) / `dq maintain` (snapshot maintenance)
#   integration.test - the stand-level tests executed by the `verify` service
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY tests ./tests
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dq ./cmd/dq
RUN CGO_ENABLED=0 GOOS=linux go test -c -o /out/integration.test ./tests

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 app
USER app
COPY --from=build /out/dq /usr/local/bin/dq
COPY --from=build /out/integration.test /usr/local/bin/integration.test
CMD ["dq", "run"]
