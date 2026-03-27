FROM golang:1.25.6-alpine3.23 as builder

WORKDIR /app

COPY . .

ENV CGO_ENABLED=0 \
    GO_PROXY=https://goproxy.cn,direct

RUN go mod tidy && \
    go build -o k8s-manage .

FROM alpine:3.23.3

WORKDIR /app

COPY --from=builder /app/k8s-manage .

RUN chmod +x k8s-manage

CMD [ "./k8s-manage" ]