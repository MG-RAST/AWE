# AWE worker with CWL runner

# docker build -t mgrast/awe-worker -f Dockerfile_worker .


FROM golang:1.11.1-alpine

# git needed for GIT_COMMIT_HASH
RUN apk update && apk add git


# install cwl-runner with node.js
RUN apk update ; apk add python3-dev nodejs gcc musl-dev libxml2-dev libxslt-dev py3-libxml2 py3-psutil   
RUN pip3 install --upgrade pip ; pip3 install cwltool ; ln -s /usr/bin/cwltool /usr/bin/cwl-runner      # cwl-runner was pointing to old version 



ENV AWE=/go/src/github.com/MG-RAST/AWE
WORKDIR /go/bin

COPY . /go/src/github.com/MG-RAST/AWE

# backwards compatible pathing with old dockerfile
RUN ln -s /go /gopath

# compile AWE
RUN mkdir -p ${AWE} && \
  cd ${AWE} && \
  go get -d ./awe-worker/ && \
  ./compile-worker.sh

