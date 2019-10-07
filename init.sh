#!/bin/bash

DOCKER_VERSION=$(docker --version | grep -o "[0-9]*\.[0-9]*\.[0-9a-z\.-]*")
echo "DOCKER_VERSION: ${DOCKER_VERSION}"
BIN_DIR="tmp/bin"
mkdir -p ${BIN_DIR}
export DOCKER_BINARY="${PWD}/${BIN_DIR}/docker-${DOCKER_VERSION}"
echo "DOCKER_BINARY: ${DOCKER_BINARY}"
if [ ! -e ${DOCKER_BINARY} ] ; then
  mkdir -p ${BIN_DIR}
  cd ${BIN_DIR}
  curl --progress -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz
  tar -xvzf docker-${DOCKER_VERSION}.tgz  docker/docker
  mv docker/docker ${DOCKER_BINARY}
fi

export DATADIR=${PWD}/data
echo "DATADIR: ${DATADIR}"
