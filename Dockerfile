FROM golang:latest AS build
WORKDIR /go/src
COPY . /go/src
RUN go build ./cmd/testing

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /root/
COPY --from=build /go/src/testing .
EXPOSE 8080
CMD ["./testing", "http"]
