

Start test environment
```bash
docker-compose up
```


## Container to submit CWL compliance tests
```bash
export DATADIR=${HOME}/awe_data  # use whatever directory your data is located in
docker rm -f awe-submitter > /dev/null 2>&1
docker run -ti --network awe-test_default --name awe-submitter -e SHOCK_SERVER="http://shock:7445" -e AWE_SERVER="http://awe-server:80" --workdir=/go/src/github.com/MG-RAST/AWE -v ${DATADIR}:${DATADIR}  -v ${HOME}/gopath/src:/go/src --entrypoint=/bin/ash mgrast/awe-submitter-testing
```


Compile if needed
```bash
cd $AWE ; ./compile-submitter.sh
```



Submit compliance test #1
```bash
cd /common-workflow-language/ ; ./run_test.sh --junit-xml=/output/result.xml --timeout=120 -n1 RUNNER=/go/bin/awe-cwl-submitter-wrapper.sh
```




# Development
The AWE test environment can also be used as a development environment. The approach below allows a developer to replace a container in the docker-compose environment with their own interactive container that has their source code mounted, which makes it easy to compile and test awe code.

## AWE server development
Start development container 
```bash
docker rm -f awe-test_awe-server_1 > /dev/null 2>&1
docker run -ti --name awe-test_awe-server_1 --rm --network awe-test_default --network-alias awe-server -p 81:80 -v ${HOME}/gopath/src:/go/src mgrast/awe-server
```

(Inside container) Compile:
```bash
cd $AWE ; ./compile.sh
```

(Inside container) Start awe-server:
```bash
awe-server --logoutput=console --debuglevel=3 --hosts=awe-mongo --api-port=80 --api-url='http://localhost:81' --title="test AWE server" --max_work_failure=1 --recover --max_client_failure=1000
```


## AWE worker development
Download the statically compiled docker binary if missing. (This is needed by the cwl-runnner to execute CWL workflows.)
```bash
DOCKER_VERSION=$(docker --version | grep -o "[0-9]*\.[0-9]*\.[0-9a-z\.-]*")
echo "DOCKER_VERSION: ${DOCKER_VERSION}"
mkdir -p "${HOME}/bin/"
DOCKER_BINARY="${HOME}/bin/docker-${DOCKER_VERSION}"
if [ ! -e ${DOCKER_BINARY} ] ; then
  mkdir -p ${HOME}/tmp/
  cd ${HOME}/tmp/
  curl -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz
  tar -xvzf docker-${DOCKER_VERSION}.tgz -C ${HOME}/tmp/ docker/docker
  mv docker/docker ${DOCKER_BINARY}
fi
```

Start development container 
```bash
export NAME=awe-test_awe-worker_1
export WORKER_DATADIR=${HOME}/awe_data

docker rm -f ${NAME} > /dev/null 2>&1
docker run -ti --network awe-test_default --name ${NAME} -e NAME=${NAME} -e WORKER_DATADIR=${WORKER_DATADIR} --workdir=/go/src/github.com/MG-RAST/AWE -v ${WORKER_DATADIR}:${WORKER_DATADIR}  -v ${DOCKER_BINARY}:/usr/local/bin/docker -v /var/run/docker.sock:/var/run/docker.sock -v ${HOME}/gopath/src:/go/src -v /tmp:/tmp  mgrast/awe-worker ash
```

(Inside container) Compile:
```bash
cd $AWE ; ./compile-worker.sh
```

(Inside container) Start awe-worker:
```bash
/go/bin/awe-worker  --name ${NAME} --data=${WORKER_DATADIR}/data --logs=${WORKER_DATADIR}/logs --workpath=${WORKER_DATADIR}/work  --serverurl=http://awe-server:80 --group=default --supported_apps=* --auto_clean_dir=false --debuglevel=3 
```


