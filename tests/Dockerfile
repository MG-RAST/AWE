

# docker build -t mgrast/awe-submitter-testing .


# requires environment variables SHOCK_SERVER and AWE_SERVER, e.g.:
# -e SHOCK_SERVER="http://skyport.local:8001/shock/api/" -e AWE_SERVER="http://skyport.local:8001/awe/api/"

FROM mgrast/awe-submitter


#get compliance tests
RUN cd / ; git clone https://github.com/common-workflow-language/common-workflow-language.git
RUN apk add bash
RUN pip install cwltest

COPY awe-cwl-submitter-wrapper.sh /go/bin/awe-cwl-submitter-wrapper.sh

RUN chmod u+x /go/bin/awe-cwl-submitter-wrapper.sh

WORKDIR /common-workflow-language




