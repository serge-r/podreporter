FROM golang:1.15-alpine AS build_base
RUN apk add --no-cache git
WORKDIR /tmp/podreporter
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o ./build/podreporter .

# Start new from a smaller image
FROM alpine:latest
RUN apk add ca-certificates
COPY --from=build_base /tmp/podreporter/build/podreporter /app/podreporter
CMD ["/app/podreporter"]