FROM golang:1.20 as build

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN script/install-libtensorflow

RUN go build -a -o 'snips.sh'

FROM ubuntu:20.04

COPY --from=build /build/snips.sh /usr/bin/snips.sh
COPY --from=build /usr/local/lib/libtensorflow.so.2 /usr/local/lib/
COPY --from=build /usr/local/lib/libtensorflow_framework.so.2 /usr/local/lib/

RUN ldconfig

ENV SNIPS_HTTP_INTERNAL=http://0.0.0.0:8080
ENV SNIPS_SSH_INTERNAL=ssh://0.0.0.0:2222

EXPOSE 8080 2222

ENTRYPOINT [ "/usr/bin/snips.sh" ]
