# AWE Server with MongoDB

# The normal AWE Dockerfile will also create the AWE server binary, but it will not install MongoDB. If you want to run MongoDB inside the same container as the AWE server runs, use this Dockerfile. Note that it depends on an awe image.

# to run mongodb in the same container as the awe-server
FROM skyport/awe:latest

#to run mongodb in a separate container
#FROM ubuntu:14.10

RUN apt-get update && apt-get install -y curl

RUN mkdir -p /mongodb/ ; \
  curl -s http://downloads.mongodb.org/linux/mongodb-linux-x86_64-2.4.14.tgz | tar -v -C /mongodb/ -xz

# execute directly: 
# /mongodb/bin/mongod --dbpath /data/
# execute in background: 
# nohup /mongodb/bin/mongod --dbpath /data/ &
