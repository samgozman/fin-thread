FROM golang:1.21.1 AS builder

ARG TELEGRAM_CHANNEL_ID
ARG TELEGRAM_BOT_TOKEN
ARG OPENAI_TOKEN

WORKDIR /app
# TODO: Add a .dockerignore file
# TODO: Add split build stage for dependencies
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o finfeed

FROM gcr.io/distroless/static-debian11:latest
COPY --from=builder /app/finfeed /finfeed
WORKDIR /app

ENV TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}
ENV TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
ENV OPENAI_TOKEN=${OPENAI_TOKEN}

CMD ["/finfeed"]