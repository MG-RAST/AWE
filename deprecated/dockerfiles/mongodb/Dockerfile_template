# MongoDB


FROM ubuntu:14.10

RUN apt-get update && apt-get install -y curl

RUN mkdir -p /mongodb/ ; \
  curl -s http://downloads.mongodb.org/linux/mongodb-linux-x86_64-[% version %].tgz | tar -v -C /mongodb/ -xz


RUN ln -s /mongodb/mongodb-linux-x86_64-[% version %]/bin/mongod /usr/bin/mongod
ENV PATH $PATH:/mongodb/mongodb-linux-x86_64-[% version %]/bin

CMD ["mongod"]

# execute directly: 
# /mongodb/bin/mongod --dbpath /data/
# execute in background: 
# nohup /mongodb/bin/mongod --dbpath /data/ &

