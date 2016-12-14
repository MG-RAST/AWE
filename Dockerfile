#export TAG=`date +"%Y%m%d.%H%M"`
#export NAME=mgrast/awe
#docker build --force-rm --no-cache --rm -t ${NAME}:${TAG} .

#FROM golang:1.7.4-alpine
FROM golang:1.7.4-wheezy

# needed for GIT_COMMIT_HASH
#RUN apk update && apk add git
RUN apt-get update && apt-get install -y libsasl2-dev

ENV AWE=/go/src/github.com/MG-RAST/AWE
WORKDIR /go/bin

COPY . /go/src/github.com/MG-RAST/AWE

# backwards compatible pathing with old dockerfile
RUN ln -s /go /gopath

# compile AWE
RUN mkdir -p ${AWE} && \
  cd ${AWE} && \
  GITHASH=$(git rev-parse HEAD) && \
  go install -a -v -race -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ...
#  GITHASH=$(git -C ${AWE} rev-parse HEAD) && \
#  CGO_ENABLED=0 go install -a -installsuffix cgo -v -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ...

# since this produces three binaries, we just specify (b)ash
CMD ["/bin/bash"]
#CMD ["/bin/ash"]
