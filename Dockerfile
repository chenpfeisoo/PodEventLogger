FROM golang:1.15 as build

WORKDIR /go/src/PodEventLogger
COPY . .

RUN go install -v PodEventLogger

FROM debian:stretch-slim

COPY --from=build /go/bin/PodEventLogger /usr/bin/PodEventLogger

ENTRYPOINT ["PodEventLogger"]
