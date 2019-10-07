#!/bin/bash




set -x
AWE_SERVER_HELP=$(docker run -i --rm mgrast/awe-server:latest /go/bin/awe-server --fullhelp | tr '\n' '%')

AWE_WORKER_HELP=$(docker run -i --rm mgrast/awe-worker:latest /go/bin/awe-worker --fullhelp | tr '\n' '%')

AWE_SUBMITTER_HELP=$(docker run -i --rm mgrast/awe-submitter:latest /go/bin/awe-submitter --fullhelp | tr '\n' '%')
set +x



sed -e "s@\[AWE-SERVER-HELP\]@${AWE_SERVER_HELP}@" -e "s@\[AWE-WORKER-HELP\]@${AWE_WORKER_HELP}@"  -e "s@\[AWE-SUBMITTER-HELP\]@${AWE_SUBMITTER_HELP}@"  ./config_template.md | tr '%' '\n' > ./config.md