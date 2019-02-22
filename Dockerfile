FROM golang:1.11-alpine as builder

RUN apk add --update make
WORKDIR /go/src/github.com/xrstf/bunker/
COPY . .
RUN make

FROM alpine:3.8

RUN apk --no-cache add
WORKDIR /app
COPY --from=builder /go/src/github.com/xrstf/bunker/bunker .
ENTRYPOINT ["./bunker"]
