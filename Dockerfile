FROM golang:1.20 as build

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN script/install-libtensorflow

RUN go build -a -o 'snips.sh'

# using ubuntu instead of something smaller like alpine so we don't need to deal with musl
# thanks tensorflow https://github.com/tensorflow/tensorflow/issues/15563#issuecomment-353797058
FROM ubuntu:20.04

RUN apt update && apt install -y curl

COPY --from=build /build/snips.sh /usr/bin/snips.sh
COPY --from=build /build/script/install-libtensorflow /tmp/install-libtensorflow

RUN /tmp/install-libtensorflow && rm /tmp/install-libtensorflow

CMD [ "/usr/bin/snips.sh" ]
