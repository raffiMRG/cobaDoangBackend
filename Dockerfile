# Gunakan image Golang resmi sebagai builder
FROM golang:1.23 AS builder

# Set workdir di dalam container
WORKDIR /app

# Copy file go.mod dan go.sum lalu download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy semua file source code
COPY . .

# Build aplikasi (output binary ke /app/app)
RUN go build -o app .

# Gunakan image minimal untuk menjalankan binary
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary dari builder
COPY --from=builder /app/app .

# Jalankan binary
CMD ["./app"]
