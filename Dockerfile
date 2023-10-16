FROM golang:1.20-alpine

ENV SRC_DIR /netflux
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

COPY . $SRC_DIR
RUN cd $SRC_DIR \
  && go mod download

RUN cd $SRC_DIR \
    && go mod download \
    && go build -o main

EXPOSE 3721 3824

CMD cd $SRC_DIR && ./main