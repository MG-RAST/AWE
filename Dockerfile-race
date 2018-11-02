#export TAG=`date +"%Y%m%d.%H%M"`
#export NAME=mgrast/awe
#docker build --force-rm --no-cache --rm -t ${NAME}:${TAG} .

FROM golang:1.11.1

# needed for GIT_COMMIT_HASH
RUN apt-get update && apt-get install -y libsasl2-dev

ENV AWE=/go/src/github.com/MG-RAST/AWE
WORKDIR /go/bin

COPY . /go/src/github.com/MG-RAST/AWE

# backwards compatible pathing with old dockerfile
RUN ln -s /go /gopath

# compile AWE
RUN mkdir -p ${AWE} && \
  cd ${AWE} && \
  GITHASH=$(git -C ${AWE} rev-parse HEAD) && \
  go install -a -v -race -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ./awe-worker ./awe-server

# since this produces two binaries, we just specify (b)ash
CMD ["/bin/bash"]
