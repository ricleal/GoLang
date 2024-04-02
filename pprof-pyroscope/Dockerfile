# syntax=docker/dockerfile:1

###############
# Build stage #
###############
FROM golang:1.22.1-bullseye as builder

WORKDIR /app

# Add source code
COPY main.go ./

COPY go.mod ./

# Build
RUN go build -o /app/app .


#################
# Runtime stage #
#################

FROM ubuntu:22.04

ENV API_PORT=8081

COPY --from=builder /app/app /app/

ENTRYPOINT /app/app -port ${API_PORT}
