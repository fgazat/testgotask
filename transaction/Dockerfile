FROM golang:1.22-alpine as builder
WORKDIR /app
COPY . ./
RUN go mod download
RUN go mod tidy
RUN go build -o transaction-service
RUN > /app/.env

# Use a minimal image for the final stage
FROM scratch
WORKDIR /root/
COPY --from=builder /app/transaction-service .
COPY --from=builder /app/.env .

EXPOSE 8082
CMD ["./transaction-service"]
