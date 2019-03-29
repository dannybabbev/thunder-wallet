FROM golang:1.11.0-stretch

WORKDIR /go/src/app

ADD . /go/src/app

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure && go build

EXPOSE 8888

CMD ./app