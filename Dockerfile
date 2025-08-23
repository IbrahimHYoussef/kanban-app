    # Stage 1: Build the Go application
    FROM golang:1.23-alpine AS builder   
    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    RUN CGO_ENABLED=0 GOOS=linux go build -o myapp .

    # Stage 2: Create the final, lightweight image
    FROM scratch
    WORKDIR /app
    COPY --from=builder /app/myapp .
    COPY . .
    
    EXPOSE 4000 
    # Expose the port your application listens on
    CMD ["./myapp --mode prod"]