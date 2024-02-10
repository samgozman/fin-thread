FROM golang:1.22.0-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o finfeed

FROM gcr.io/distroless/static-debian12:latest
COPY --from=builder /app/finfeed /finfeed
WORKDIR /app
CMD ["/finfeed"]