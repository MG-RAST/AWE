##Create Docker images

Docker images can be created either with Dockerfiles or manually by making a snapshot of a container.

### Dockerfiles
Dockerfiles contain installation instructions (similar to a Bash script) that are used to build Docker images. An example Dockerfile can be found here (a link with more example is provided below): 

[https://github.com/MG-RAST/Skyport/blob/master/dockerfiles/velvet/Dockerfile](https://github.com/MG-RAST/Skyport/blob/master/dockerfiles/velvet/Dockerfile)

Each RUN instruction is executed in its own container. The filesystem modifications of each container are saved in an image layer. The resulting image from this build process consists of multiple layers representing the history of the build process.

A simple example to install velvet from the ubuntu repository:

```shell
docker build -t skyport/velvet:1.2.10 https://raw.githubusercontent.com/MG-RAST/Skyport/master/dockerfiles/velvet/Dockerfile
```

Other examples for such Dockerfiles can be found here:

[https://github.com/MG-RAST/Skyport/tree/master/dockerfiles](https://github.com/MG-RAST/Skyport/tree/master/dockerfiles)

Full documentation for Dockerfiles from Docker:

[https://docs.docker.com/reference/builder/](https://docs.docker.com/reference/builder/)

### Snapshot of a container

```shell
host> docker run -t -i ubuntu:14.04 /bin/bash
root@container_id> apt-get update
root@container_id> apt-get install -y velvet
root@container_id> exit
host> docker commit <your container_id> skyport/velvet:1.2.10
```
In this example the user creates a new ubuntu container. The user installs and configures everything that is needed to run his application in this container. After exiting, he can make snapshot of this container using the "docker commit" command.

Images that were created with "docker commit" consist only of one layer.

### Dockerfile vs. Snapshot
Builds from Dockerfiles are layered. Each layer contains only changes to the previous layer. If a top layer deletes a files, that file is not visible in the container, but it is still exiting in the previous layer, thus wasting hard-drive space. This is something to consider when Dockerfile is created. The advantage of layered images is that multiple images can share the same base layer (e.g. the ubuntu layer). Also, when a new layered image is created, it is possible to reuse existing layers, which speeds up the build process. The Dockerfile documents how an image was created and can be helpful to create new versions of an image. However, a Dockerfile does not guarantee reproducibility, as it can contain external dependencies (e.g. "apt-get update" or "git clone ...") which will yield different results at different points in time. 


##Upload Docker Images to Shock

To upload docker image to Shock, please use the perl script provided here:

[https://github.com/MG-RAST/Skyport](https://github.com/MG-RAST/Skyport)

Please note that by default the uploaded image will be public. If the Shock server uses authentication you will need to get an authentication token from the administrator. Within the KBase project this will be the the KB_AUTH_TOKEN.

**Technical details**

Docker images can be uploaded to Shock also with curl, but in order usable by AWE workers, they must have certain metadata:

```shell
"attributes": {
    "base_image_id": "",
    "base_image_tag": "",
    "docker_version": {
        "ApiVersion": "1.13",
        "Arch": "amd64",
        "GitCommit": "bd609d2",
        "GoVersion": "go1.2.1",
        "KernelVersion": "3.13.0-29-generic",
        "Os": "linux",
        "Version": "1.1.1"
    },
    "dockerfile": "",
    "id": "0573f572bf745fb261e4b6041010b4cd4d971c7acb85c6f0ecbfe826f98876f2",
    "name": "skyport/bowtie2:2.1.0",
    "type": "dockerimage"
```

Example to export image from docker:

```shell
docker save skyport/velvet:1.2.10 > skyport_velvet1.2.10.tar
```

The Skyport Perl script mentioned above does this steps automatically. It saves the image, uploads the image and saves the metadata.