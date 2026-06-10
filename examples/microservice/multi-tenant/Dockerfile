# Start from the latest golang base image
FROM golang:latest AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY main.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o main .


######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Create a non-root system user named 'app'
RUN adduser -S app

# Command to run the executable
USER app
CMD ["./main"]
