FROM golang:latest AS build
WORKDIR /go/src
COPY . /go/src
RUN go build ./cmd/tekton-ci

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /root/
COPY --from=build /go/src/tekton-ci .
EXPOSE 8080
ENTRYPOINT ["./tekton-ci"]
