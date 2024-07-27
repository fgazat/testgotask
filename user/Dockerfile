FROM golang:1.22-alpine as builder
WORKDIR /app
COPY . ./
RUN go mod download
RUN go mod tidy
RUN go build -o users-service
RUN > /app/.env

# Use a minimal image for the final stage
FROM scratch
WORKDIR /root/
COPY --from=builder /app/users-service .
COPY --from=builder /app/.env .

EXPOSE 8081
CMD ["./users-service"]
