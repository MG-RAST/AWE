
build=$1
echo Setup build in ${build}
cd $1

export TAG=CWL

git clone https://github.com/MG-RAST/Skyport2.git
git clone https://github.com/MG-RAST/AWE.git
git clone https://github.com/common-workflow-language/common-workflow-language.git
git clone https://github.com/common-workflow-language/cwltest.git

# Build cwltest
cd cwltest/
mkdir install
export PYTHONPATH=$PATHONPATH:`pwd`/install//lib/python2.7/site-packages/
export PATH=$PATH:`pwd`/install/bin/
python setup.py install --prefix `pwd`/install/
cd .. 

### Build ne AWE container

#git clone https://github.com/MG-RAST/Skyport2.git
### Build ne AWE container
export TAG=CWL
cd AWE
docker build -t mgrast/awe:${TAG} -f Dockerfile .
docker build -t mgrast/awe-worker:${TAG} -f Dockerfile_worker .
docker build -t mgrast/awe-submitter:${TAG} -f Dockerfile_submitter .

### Copy submitter
docker create --name submitter mgrast/awe-submitter:${TAG}
docker cp submitter:/go/bin/awe-submitter awe-submitter
docker rm submitter 

### Start stack
cd ../Skyport2	
./init.sh
. ./skyport2.env
# skyport2.env overwrites TAG - set back
export TAG=CWL
docker-compose down
docker-compose -f Docker/Compose/skyport-awe-jenkins.yaml up -d
sleep 60

# Execute tests
cd ../common-workflow-language/
./run_test.sh RUNNER=`pwd`/../AWE/tests/awe-cwl-submitter-wrapper.sh

# Shut down
cd ../Skyport2
docker-compose -f Docker/Compose/skyport-awe-jenkins.yaml  down
## TEST COMES HERE
