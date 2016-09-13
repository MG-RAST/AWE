#[build] export TAG=`date +"%Y%m%d.%H%M"`
#[build] export NAME=mgrast/awe
#[build] docker build -t ${NAME}:${TAG} .

FROM golang:1.6.3-alpine

# needed for GIT_COMMIT_HASH
RUN apk update && apk add git

ENV AWE=/go/src/github.com/MG-RAST/AWE
WORKDIR /go/bin

COPY . /go/src/github.com/MG-RAST/AWE

# compile AWE
RUN mkdir -p ${AWE} && \
  cd ${AWE} && \
  GITHASH=$(git -C ${AWE} rev-parse HEAD) && \
  CGO_ENABLED=0 go install -a -installsuffix cgo -v -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ...

# since this produces three binaries, we just specify (b)ash
CMD ["/bin/ash"]
