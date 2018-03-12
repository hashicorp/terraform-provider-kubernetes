FROM golang:1.10
WORKDIR /go/src/github.com/terraform-providers/terraform-provider-kubernetes/
COPY ./ /go/src/github.com/terraform-providers/terraform-provider-kubernetes/
RUN go get -v
RUN scripts/build.sh