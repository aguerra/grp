FROM golang:1.7

RUN mkdir -p /go/src/github.com/aguerra/grp
WORKDIR /go/src/github.com/aguerra/grp
COPY . /go/src/github.com/aguerra/grp

RUN make install

ENTRYPOINT ["grp"]
EXPOSE 2083/tcp
