


# AWE 2.0 

 A light weight workflow manager for scientific computing at scale that:

- executes [CWL](http://www.commonwl.org) workflows

- is a cloud native workflow platform designed from the ground up to be fast, scalable and fault tolerant.

- supports [CWLprov](https://github.com/common-workflow-language/cwlprov)

- is RESTful. The [API](./API/README.md) is accessible from desktops, HPC systems, exotic hardware, the cloud and your smartphone.

- is designed for complex scientific computing and supports computing on multiple platforms with zero setup.

- supports containerized environments with Docker and Singularity

- is part of our reproducible science platform [Skyport]([https://github.com/MG-RAST/Skyport2) combined to create [Researchobjects](http://www.researchobject.org/) when combined with [CWL](http://www.commonwl.org) and 
[CWLprov](https://github.com/common-workflow-language/cwlprov)

AWE is actively being developed at [github.com/MG-RAST/AWE](https://github.com/MG-RAST/AWE).


You can use AWE simultaenously on Clouds, Clusters and HPC systems with dozens, hundreds or thousands of nodes to run tens of thousands to hundreds of thousands of individuals workflows. 


Check out the notes  on [building and installing](./building.md) and [configuration](./configuration.md).


## AWE in 30 seconds (for Docker-compose) (or later for kubectl )
This assumes that you have `docker` and `docker-compose` installed and `curl` is available locally.

### setup the local service

```bash
source ./init.sh
docker-compose up
```

Don't forget to later `docker-compose down` and do not forget, by default this demo configuration does not store data persistently.


### Submit a job for simple workflow
~~~~
a simple workflow in CWL to create a wordle PDF from a PDF
~~~~

### View result
~~~~
open image with wordle/ preview here
~~~~

## Documentation
- [API documentation](./API.md).
- [Building](./building.md).
- [Configuring](./config.md).
- [Concepts](./concepts.md).
- [Caching and data migration](./caching_and_data_migration.md).
- For further information about Shock's functionality, please refer to our [Shock documentation](https://github.com/MG-RAST/Shock/docs/).



