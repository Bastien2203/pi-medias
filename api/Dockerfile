# Build stage
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o media-api .

# Final stage
FROM scratch
COPY --from=builder /app/media-api /media-api
# The /media directory will be mounted as a volume from docker-compose.
EXPOSE 8080
CMD ["/media-api"]
