# syntax=docker/dockerfile:1.6
FROM golang:1.27.1-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

# CGO_ENABLED=0 binary: distroless/static is enough. Runs as uid/gid 65532.
FROM gcr.io/distroless/static-debian12:nonroot

ENV APP_ENV=production
ENV PORT=8080

COPY --from=builder /bin/api /bin/api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/bin/api"]
