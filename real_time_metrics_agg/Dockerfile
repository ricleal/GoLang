FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./real_time_metrics_agg/server

FROM scratch
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
