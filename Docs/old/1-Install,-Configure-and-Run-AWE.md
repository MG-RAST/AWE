
The recommended installation method for AWE is with Docker. The AWE-binaries in the AWE Docker images are statically compiled, which means binaries can be copied from the Docker images and executed without Docker containers. Instructions are in Section 2.

# 1. Install Docker
See Docker website for platform specific instructions:

[https://www.docker.com/get-docker](https://www.docker.com/get-docker)

If you prefer a manual installation please use our Dockerfiles for instructions. Note that these Dockerfiles create statically compiled binaries that you can easily extract. (Also note that the AWE worker binary is named awe-worker.) 

To install Docker, e.g. using ubuntu:

```bash
sudo apt-get update && sudo apt-get -y install docker.io
```

**Optional:**<br>Docker requires quite some disk space, you can set these environment variables to tell docker to use ephemeral disk space in the docker config file:
```bash
# /etc/default/docker.io
export TMPDIR=/media/ephemeral/tmp/
export DOCKER_OPTS="-g=/media/ephemeral/docker"
```

Create tmp dir:
```bash
mkdir /media/ephemeral/tmp/
```
And restart docker: 
```bash
sudo service docker.io restart
```

# 2. Get AWE Docker image / binaries
Download AWE image from DockerHub:
```bash
docker pull mgrast/awe
docker pull mgrast/awe-worker
docker pull mongo # only needed for AWE server
```

## Optional: Build image from Dockerfile
You can also build the image on your own from the Dockerfiles. 

Example to build image:
```shell
export TAG=`date +"%Y%m%d.%H%M"`
git clone --recursive https://github.com/MG-RAST/AWE.git
cd AWE
docker build --no-cache -t mgrast/awe:$TAG .
```
## Optional: Binaries
If you need binaries, either download binaries for the latest tagged version from github [https://github.com/MG-RAST/AWE/releases/latest)](https://github.com/MG-RAST/AWE/releases/latest), or extract the very latest version from the Dockerimages:

```bash
docker create --name awe-temporary mgrast/awe-server
docker cp awe-temporary:/go/bin/awe-server .
docker rm awe-temporary

docker create --name awe-temporary mgrast/awe-worker
docker cp awe-temporary:/go/bin/awe-worker .
docker rm awe-temporary
```


# 3. Run AWE
## 3.1 AWE worker
This example assumes you run the AWE worker in a container. If you do not use Docker, ignore the line that starts with "docker run". The AWE worker will store files on the host in the DATADIR directory. Do not forget to specifiy the AWE server url. Here the AWE worker is completely configured by command line parameters and does not need a configuration file.  This command will run AWE worker as a demonized container:

```bash
export DATADIR=/media/ephemeral/awe-worker/
mkdir -p ${DATADIR}
docker run -d --name awe-worker -v /var/run/docker.sock:/var/run/docker.sock -v ${DATADIR}:${DATADIR}  mgrast/awe \
/gopath/bin/awe-worker --data=${DATADIR}/data --logs=${DATADIR}/logs --workpath=${DATADIR}/work  --serverurl=<awe_server> --group=docker --supported_apps=*
```

To run in foreground replace "-d" with "-t -i --rm" 
```bash
docker run -t -i --rm --name ...
```

Verify the container is running:
```bash
sudo docker ps
```


## 3.2 AWE server

The AWE server requires a Mongodb instance. On ubuntu the mongo server should already be running once installed. With Docker you can start an instance like this:

```bash
export DATADIR="/media/ephemeral/awe-server/"
docker run --rm --name awe-server-mongodb -v ${DATADIR}mongodb/:/data/db --expose=27017 mongo mongod --dbpath /data/db
```

You can configure the AWE server with a configuration file and via command line options. You can start with a template config file for the AWE server and then make changes: 
```bash
/usr/bin/curl -o ${DATADIR}awe-server.cfg https://raw.githubusercontent.com/MG-RAST/AWE/master/templates/awes.cfg.template
```

This is the command to start the AWE server. If you do not use Docker, ignore the line that starts with "docker run":
```bash
export TITLE="my server"
export PUBLIC_IPV4=<PUBLIC IP>
docker run --rm --name awe-server -p 80:80 -p 8001:8001 -v ${DATADIR}awe-server.cfg:/awe-config/awe-server.cfg -v ${DATADIR}data/:/mnt/data/awe/ --link=awe-server-mongodb:mongodb mgrast/awe \
/gopath/bin/awe-server --hosts=mongodb --debuglevel=1 --conf=/awe-config/awe-server.cfg --use_app_defs=yes --site-port=80 --api-port=8001 --api-url=http://${PUBLIC_IPV4}:8001 --site-url=http://${PUBLIC_IPV4}:80 --title="${TITLE}"
```
Comments:<br>
port 80: AWE server monitor<br>
port 8001: AWE server API<br>
"-v" mounts host to container directories <br>
"--link" connects AWE server and mongodb

If you run awe server binary directly, note that you will need the "site" sub-directory in the AWE git repository (https://github.com/MG-RAST/AWE/tree/master/site) to use the AWE monitor.


### awe-server configuration file template:

https://raw.githubusercontent.com/MG-RAST/AWE/master/templates/awes.cfg.template

Config file specification is available at, or use "awe-server --help"

https://github.com/MG-RAST/AWE/wiki/AWE-config-file-manual



AWE options:	
--recover: if this flag is set,	awe-server will	automatically recover the queue before last time it was shut down.
--debug:	set the	debug log level	from 0	to 3, by default debug log is not printed
--dev: print simple queue statistics to stdout every 10 seconds
	

Check logs:	
* Log location: \<path/to/awe/logs\> configured in config file.
* Log types: access.log, error.log, debug.log, event.log, perf.log


# 4. AWE Web Monitor
TODO: this requires update, this is partly automatic now.

# 4.1 Configure the web interfaces and monitor jobs with web monitor
**WARNING**: Soon deprecated. The default configuration will be automatically set up and does only require the port to be defined on which the AWE Monitor listens.

1) Configure the web site path in awe.cfg for awe-server

e.g.
 
    [Directories]
    Site=/Users/wtang/gopath/src/github.com/MG-RAST/AWE/site   #where the js are
    [Ports]
    site-port=8080    #what port you want to use as the web interface

2) Create a config file (config.js) under server machine $GOPATH/src/MG-RAST/AWE/site/js

you can rename config.js.tt to config.js, and edit the content of config.js:

e.g.

    var RetinaConfig = {
        "awe_ip": "http://140.221.84.148:8000",
        "workflow_ip": "http://140.221.84.148:8000/awf"
    }

* 140.221.84.148:8000 is the external server API URL. 
* After above configuration, your web interface can be accessed by: http://140.221.84.148:8080/
* You can use localhost as the ip address, but then you can only access the web interface from the local machine by http://localhost:8080


# 5 Shock
For installation and configuration of Shock, please refer to https://github.com/MG-RAST/Shock



# Notes
To update vendorized code in AWE
```bash
cd AWE
go get -v github.com/mjibson/party
party -d vendor -c -u
```

# User-data file for VMs
Can be used to conveniently start multiple AWE worker instance in the cloud. Requires recent Ubuntu image.

```bash
#!/bin/bash
set -e
set -x

# configuration
export AWE_SERVER="<server-address:port>" # include http://
export GROUP="docker"
export EPHEMERAL="/mnt"
# or export EPHERMERAL="/media/ephemeral/"
export DATADIR=${EPHEMERAL}/awe-worker/
export WORKERNAME=""

# Detect IP and set worker name
# export WORKERNAME=`ifconfig eth0 | grep 'inet addr' | cut -d ":" -f 2 |cut -d " " -f1`

# install docker
apt-get update && sudo apt-get -y install docker.io
sleep 2

# move docker location to EPHEMERAL
# the default location for Docker to store images may not have enough space:
mkdir -p ${EPHEMERAL}/tmp/
echo 'export TMPDIR='${EPHEMERAL}'/tmp/' >> /etc/default/docker.io
echo 'export DOCKER_OPTS="-g='${EPHEMERAL}'/docker"' >> /etc/default/docker.io
/usr/sbin/service docker.io restart
sleep 4

# Create directory for awe-worker data:
mkdir -p ${DATADIR}

# Run awe-worker:
/usr/bin/docker run -d --name awe-worker -v /var/run/docker.sock:/var/run/docker.sock -v ${DATADIR}:${DATADIR}  mgrast/awe /gopath/bin/awe-worker --data=${DATADIR}/data --logs=${DATADIR}/logs --workpath=${DATADIR}/work  --serverurl=${AWE_SERVER} --group=${GROUP} --debuglevel=1 --auto_clean_dir=false --supported_apps=*

#Start awe-worker on reboots. This step is optional (better would be upstart or systemd unit), thus it is commented:
#echo "/usr/bin/docker run -d --name awe-worker -v /var/run/docker.sock:/var/run/docker.sock -v ${DATADIR}:${DATADIR}  mgrast/awe /gopath/bin/awe-worker --data=${DATADIR}/data --logs=${DATADIR}/logs --workpath=${DATADIR}/work  --serverurl=${AWE_SERVER} --name=${WORKERNAME}--group=${GROUP} --debuglevel=1 --auto_clean_dir=false --supported_apps=*" >> /etc/rc.local
```
