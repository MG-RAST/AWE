
# docker build --force-rm --no-cache --rm -t mgrast/awe-server .

# binaries
# docker create --name awe-temporary mgrast/awe-server
# docker cp awe-temporary:/go/bin/awe-worker .
# docker cp awe-temporary:/go/bin/awe-submitter .
# docker rm awe-temporary


FROM golang:1.11.1-alpine

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
  go get -d ./awe-server/ && \
  ./compile.sh -a


CMD ["/bin/ash"]
