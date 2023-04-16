FROM golang:1.20 as build

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN go build -a -o 'snips.sh'

FROM gcr.io/distroless/base-debian11

COPY --from=build /build/snips.sh /

CMD [ "/snips.sh" ]
