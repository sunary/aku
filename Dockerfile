FROM public.ecr.aws/docker/library/golang:1.19.9-alpine3.18 as builder

WORKDIR /app
RUN apk update && apk upgrade && \
    apk add bash git openssh gcc libc-dev
COPY ./go.mod ./go.sum ./

ENV GOSUMDB off
RUN go mod download

COPY ./ ./
RUN go build -o /dist/server cmd/server/*.go

FROM public.ecr.aws/docker/library/alpine:3.18.0

RUN apk add --update ca-certificates tzdata curl pkgconfig && \
    rm -rf /var/cache/apk/*

COPY --from=builder /dist/server /app/bin/server

WORKDIR /app/bin
CMD ["/app/bin/server"]