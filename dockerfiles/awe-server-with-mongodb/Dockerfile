# AWE Server with MongoDB

# The normal AWE Dockerfile will also create the AWE server binary, but it will not install MongoDB. If you want to run MongoDB inside the same container as the AWE server runs, use the Dockerfile. Note that it depends on an awe image.

FROM skyport/awe:latest

RUN apt-get update && apt-get install -y mongodb-server && \
	rm -f /usr/sbin/policy-rc.d #do not start mongo db automatically