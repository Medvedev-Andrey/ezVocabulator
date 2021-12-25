FROM golang:latest as build
#   Setup working directory and copy sources
WORKDIR /app
#   Restore all required dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download
#   Copy all sources to working directory
COPY *.go .
#   Build the bot
RUN CGO_ENABLED=0 GOOS=linux go build -o /telegram_bot

FROM alpine:latest as publish
#   Mailcap adds mime detection and ca-certificates help with TLS
RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app
#   Setup user and working directory and copy sources
USER app
WORKDIR /app
#   Copy bot
COPY --from=build /telegram_bot .
#   Run the bot
CMD [ "/app/telegram_bot" ]
