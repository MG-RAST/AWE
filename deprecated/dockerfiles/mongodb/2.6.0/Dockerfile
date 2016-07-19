# MongoDB


FROM ubuntu:14.10

RUN apt-get update && apt-get install -y curl

RUN mkdir -p /mongodb/ ; \
  curl -s http://downloads.mongodb.org/linux/mongodb-linux-x86_64-2.6.0.tgz | tar -v -C /mongodb/ -xz


RUN ln -s /mongodb/mongodb-linux-x86_64-2.6.0/bin/mongod /usr/bin/mongod
ENV PATH $PATH:/mongodb/mongodb-linux-x86_64-2.6.0/bin

CMD ["mongod"]

# execute directly: 
# /mongodb/bin/mongod --dbpath /data/
# execute in background: 
# nohup /mongodb/bin/mongod --dbpath /data/ &

