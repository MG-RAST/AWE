FROM mgrast/awe-submitter:test
RUN cd / ; git clone https://github.com/common-workflow-language/common-workflow-language.git
RUN apk add bash
RUN pip install cwltest 
RUN cd $AWE ; CGO_ENABLED=0 go install  ./awe-submitter/
RUN echo '#!/bin/bash' > /go/bin/awe-cwl-submitter-wrapper.sh
RUN echo '/go/bin/awe-submitter --pack --wait \
--shockurl="http://shock:7445" \
--serverurl="http://awe-server:8001" --download_files=true $@' >> /go/bin/awe-cwl-submitter-wrapper.sh
RUN chmod u+x /go/bin/awe-cwl-submitter-wrapper.sh
WORKDIR /common-workflow-language
ENTRYPOINT [ "./run_test.sh" , "RUNNER=/go/bin/awe-cwl-submitter-wrapper.sh" ]          