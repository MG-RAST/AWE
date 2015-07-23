# AWE worker and server binaries

FROM ubuntu:14.04

RUN apt-get update && apt-get install -y \
	git-core \
	bzr \
	make \
	gcc \
	mercurial \
	ca-certificates \
	curl


RUN curl -s https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar -v -C /usr/local -xz

ENV GOROOT /usr/local/go
ENV PATH /usr/local/go/bin:/gopath/bin:/usr/local/bin:$PATH
ENV GOPATH /gopath/

# clone first to get commit hash
RUN /bin/mkdir -p /gopath/src/github.com/MG-RAST/ && \
  cd /gopath/src/github.com/MG-RAST/ && \
  git clone --recursive https://github.com/MG-RAST/AWE.git && \
  cd AWE && \
  git checkout develop

RUN GITHASH=$(git -C /gopath/src/github.com/MG-RAST/AWE rev-parse HEAD) && \
  CGO_ENABLED=0 go install -a -installsuffix cgo -v -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH ${GITHASH}" github.com/MG-RAST/AWE/...

  # dynamic: go install -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH ${GITHASH}" github.com/MG-RAST/AWE/...

CMD ["bash"]
