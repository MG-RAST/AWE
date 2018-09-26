

# docker build -t mgrast/awe-submitter-testing .


FROM mgrast/awe-submitter


#get compliance tests
RUN cd / ; git clone https://github.com/common-workflow-language/common-workflow-language.git
RUN apk add bash
RUN pip install cwltest

COPY awe-cwl-submitter-wrapper.sh /go/bin/awe-cwl-submitter-wrapper.sh

RUN chmod u+x /go/bin/awe-cwl-submitter-wrapper.sh

WORKDIR /common-workflow-language




