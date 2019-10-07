#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
${DIR}/awe-submitter --wait \
        --group=default \
        --shockurl=${SHOCK_SERVER} \
        --serverurl=${AWE_SERVER} --download_files=true $@
