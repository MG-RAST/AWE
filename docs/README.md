


# AWE 2.0 

 A light weight workflow manager for scientific computing at scale that:

- executes [CWL](http://www.commonwl.org) workflows

- is a cloud native workflow platform designed from the ground up to be fast, scalable and fault tolerant

- supports [CWLprov](https://github.com/common-workflow-language/cwlprov)

- is RESTful. The [API](./API/) is accessible from desktops, HPC systems, exotic hardware, the cloud and your smartphone

- is designed for complex scientific computing and supports computing on multiple platforms with zero setup

- supports containerized environments with Docker and Singularity

- is part of our reproducible science platform [Skyport](https://github.com/MG-RAST/Skyport2) combined to create [Researchobjects](http://www.researchobject.org/) when combined with [CWL](http://www.commonwl.org) and [CWLprov](https://github.com/common-workflow-language/cwlprov)

<br>

AWE is actively being developed at [github.com/MG-RAST/AWE](https://github.com/MG-RAST/AWE).


You can use AWE simultaenously on Clouds, Clusters and HPC systems with dozens, hundreds or thousands of nodes to run tens of thousands to hundreds of thousands of individuals workflows. 


## AWE Quickstart
This assumes that you have `docker` and `docker-compose` installed and `curl` is available locally.

### Start AWE services

```bash
source ./init.sh
docker-compose up
```
This will start AWE server, one AWE worker, Shock object store and corresponding MongoDB containers.
Don't forget to later `docker-compose down` and do not forget, by default this demo configuration does not store data persistently.


### Submit a job for simple workflow

This example consists of CWL workflow that takes a PDF file as input, extracts all words and generates a visual wordcloud.

```bash
 ./awe_submit.sh -w test/tests/pdf2wordcloud.cwl -j test/tests/rules-of-acquisition.job.cwl -d tmp
```

### View result
~~~~
open image with wordle/ preview here
~~~~

## Documentation
- [API examples](./API) and [API specification](./API/api.html)
- [Building](./building.md)
- [Configuring](./config.md)


<!--
X
- [Concepts](./concepts.md)
- [Caching and data migration](./caching_and_data_migration.md)
-->

## Related Repositories and documentation


| repository  | description                           | link                                                                     |
| ----------- | ------------------------------------- | ------------------------------------------------------------------------ |
| AWE monitor | UI for the AWE server                 | [github.com/MG-RAST/awe-monitor](https://github.com/MG-RAST/awe-monitor) |
| Shock       | object store                          | [github.com/MG-RAST/Shock](https://github.com/MG-RAST/Shock)             |
| Skyport2    | demo environment using docker-compose | [github.com/MG-RAST/Skyport2](https://github.com/MG-RAST/Skyport2)       |



