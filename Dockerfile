# AWE worker and server binaries

FROM golang:1.5.0


# old:
#FROM ubuntu:14.04
#RUN apt-get update && apt-get install -y \
#	git-core \
#	bzr \
#	make \
#	gcc \
#	mercurial \
#	ca-certificates \
#	curl
#RUN curl -s https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar -v -C /usr/local -xz
#ENV GOROOT /usr/local/go
#ENV PATH /usr/local/go/bin:/gopath/bin:/usr/local/bin:$PATH
#ENV GOPATH /gopath/


ENV AWE /go/src/github.com/MG-RAST/AWE 
RUN mkdir -p ${AWE}  ; ln -s /go /gopath
ADD . ${AWE} 
WORKDIR ${AWE}  


RUN GITHASH=$(git -C ${AWE} rev-parse HEAD) && \
  CGO_ENABLED=0 go install -a -installsuffix cgo -v -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ...


CMD ["bash"]
