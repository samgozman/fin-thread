FROM golang:1.21.1-alpine AS builder
WORKDIR /app
# TODO: Add a .dockerignore file
# TODO: Add split build stage for dependencies
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o finfeed

FROM gcr.io/distroless/static-debian11:latest
COPY --from=builder /app/finfeed /finfeed
WORKDIR /app
CMD ["/finfeed"]