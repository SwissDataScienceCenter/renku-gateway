FROM golang:1.21.6-alpine3.19 as builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/gateway cmd/gateway
COPY internal internal 
RUN go build -o /gateway github.com/SwissDataScienceCenter/renku-gateway/cmd/gateway 

FROM alpine:3.19
USER 1000:1000
COPY --from=builder /gateway /gateway
ENTRYPOINT [ "/gateway" ]

