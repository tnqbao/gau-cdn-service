FROM golang:1.25-alpine AS builder
WORKDIR /gau_cdn
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o gau-cdn-service.bin .

FROM alpine:latest
WORKDIR /gau_cdn
COPY --from=builder /gau_cdn/gau-cdn-service.bin .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
ENTRYPOINT ["./entrypoint.sh"]
