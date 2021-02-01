# BUILD STAGE
FROM golang:latest AS builder
WORKDIR /go/src/github.com/callummance/nia/
COPY . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -o nia

#RUNTIME CONTAINER
FROM alpine:latest
RUN apk add --no-cache ca-certificates bash libc6-compat
WORKDIR /root
COPY --from=builder /go/src/github.com/callummance/nia/nia .
CMD ["./nia"]