FROM golang:1.20 as build

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN script/install-libtensorflow \
    && go build -a -o 'snips.sh'

FROM alpine:3.18

COPY --from=build /build/snips.sh /usr/bin/snips.sh
COPY --from=build /usr/local/lib/libtensorflow.so.2 /usr/local/lib/
COPY --from=build /usr/local/lib/libtensorflow_framework.so.2 /usr/local/lib/

RUN apk add --no-cache libc6-compat \
    && ln -s /lib/ld-musl-x86_64.so.1 /lib/ld-linux-x86-64.so.2

ENV SNIPS_HTTP_INTERNAL=http://0.0.0.0:8080
ENV SNIPS_SSH_INTERNAL=ssh://0.0.0.0:2222

EXPOSE 8080 2222

ENTRYPOINT ["/usr/bin/snips.sh"]
