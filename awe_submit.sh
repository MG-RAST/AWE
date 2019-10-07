#!/bin/bash
# 
# skyport_submit script
#
# submit a CWL workflow, a jobinput file and a data directory for processing
# 
# simple AWE submitter

# PLEASE NOTE: all path info must be relative to DATADIR (in this case ./data)

# usage info
usage () {
        echo "Usage: skyport_submit.sh -d <datadir> -j <input.job>  -w <workflow.cwl> [-s SKYPORT_HOST]"
        echo "Options: "
        echo "              -d <datadir>          data directory"
        echo "              -j <input.job>        workflow job file,  specifies input files, yaml"
        echo "              -w <workflow.cwl>     workflow file, specifies workflow"
        echo "             [-s SKYPORT_HOST]      defaults to environment variable SKYPORT_HOST, then to localhost"
	echo "Example:  skyport_submit.sh  \ "
        echo "              -w ./CWL/Workflows/simple-bioinformatic-example.cwl \ "
        echo "              -j ./CWL/Workflows/simple-bioinformatic-example.job.yaml \ "
        echo "              -d ./CWL/Data/ "
 }

 # get options
while getopts d:w:j:s:a: option; do
    case "${option}"
        in
                w) WORKFLOW=${OPTARG};;
                j) JOBINPUT=${OPTARG};;
                d) DATADIR=${OPTARG};;
                s) SKYPORT_HOST=${OPTARG};; 
                *)
                usage
                ;;
    esac
done

USE_AUTH=""
# check on the auth situation
if [ -z ${SKYPORT_AUTH} ]
then
        echo "We did not find an auth token (-a or $SKYPORT_AUTH). Running in anonymous mode"
else
      USE_AUTH="--auth=${SKYPORT_AUTH}"
fi

# make sure the required options are present
if [ -z ${WORKFLOW} ]
then
        echo "Required parameter -w <workflow.cwl> is missing"
        usage
        exit 1
fi
# make sure the required options are present
if [ -z ${JOBINPUT} ]
then
        echo "Required parameter -j <job.yaml> is missing"
        usage
        exit 1
fi
# make sure the required options are present
if [ -z ${DATADIR} ]
then
        echo "Required parameter -d <datadir> is missing"
        usage
        exit 1
fi


if [ -s ${SKYPORT_HOST} ]
then
    SKYPORT_HOST=skyport.local
fi

echo "SKYPORT_HOST=${SKYPORT_HOST}"

WORKFLOWDIR=$(dirname ${WORKFLOW})
JOBINPUTDIR=$(dirname ${JOBINPUT})

WORKFLOW_FILE=$(basename ${WORKFLOW})
JOBINPUT_FILE=$(basename ${JOBINPUT})

if [ ! -d ${DATADIR} ]
then
 echo "directory $DATADIR does not exist!"
 exit 1
fi



#AWE_SERVER=http://${SKYPORT_HOST}:8001/awe/api/
#SHOCK_SERVER=http://${SKYPORT_HOST}:8001/shock/api/

SHOCK_SERVER=http://shock:7445
AWE_SERVER=http://awe-server:80

CURDIR=`pwd`

set -x
docker run -ti \
          --network awe-demo_default \
          --rm \
          -v ${CURDIR}/${WORKFLOWDIR}:/mnt/workflows/ \
          -v ${CURDIR}/${JOBINPUTDIR}:/mnt/jobinputs/ \
          -v ${CURDIR}/${DATADIR}:/mnt/Data/ \
          --workdir=${CURDIR}/${DATADIR} \
          mgrast/awe-submitter:latest \
          /go/bin/awe-submitter \
          --wait \
          --shockurl=${SHOCK_SERVER} \
          --serverurl=${AWE_SERVER} \
          ${USE_AUTH} \
          /mnt/workflows/${WORKFLOW_FILE} \
          /mnt/jobinputs/${JOBINPUT_FILE}
set +x


echo
