# export TAG=`date +"%Y%m%d.%H%M"`
# docker build --force-rm --no-cache --rm -t mgrast/awe .
# docker tag mgrast/awe mgrast/awe:${TAG}


FROM golang:1.7.6-alpine

#RUN apk update && apk add git gcc libc-dev cyrus-sasl-dev

# git needed for GIT_COMMIT_HASH
RUN apk update && apk add git

ENV AWE=/go/src/github.com/MG-RAST/AWE
WORKDIR /go/bin

COPY . /go/src/github.com/MG-RAST/AWE

# backwards compatible pathing with old dockerfile
RUN ln -s /go /gopath

# compile AWE
RUN mkdir -p ${AWE} && \
  cd ${AWE} && \
  go get -d ./awe-worker/ ./awe-server/ && \
  ./compile.sh

# since this produces two binaries, we just specify (b)ash
CMD ["/bin/ash"]
