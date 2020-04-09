FROM golang:latest AS build
WORKDIR /go/src
COPY . /go/src
RUN go build ./cmd/ci-hook-server

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /root/
COPY --from=build /go/src/ci-hook-server .
EXPOSE 8080
CMD ["./ci-hook-server", "http"]
