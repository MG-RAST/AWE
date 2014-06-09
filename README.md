![AWE](https://raw.github.com/wtangiit/AWE/master/site/images/awe-lg.png)
=====

About:
------

AWE is a workload management system for bioinformatic workflow applications. AWE, together with Shock data management system, can be used to build an integrated platform for efficient data analysis and management which features following functionalities:

- Explicit task parallelization and convenient application integration
- Scalable and portable workflow computation
- Integration of heterogeneous and geographically distributed computing resources
- Performance-aware, cost-efficient service management and resource management
- Reusable and reproducible data product management 

![awe-diagram](https://raw.githubusercontent.com/MG-RAST/AWE/master/site/images/awe-diagram.png)

AWE is designed as a distributed system that contains a centralized server and multiple distributed clients. The server receives job submissions and parses jobs into tasks, splits tasks into workunits, and manages workunits in a queue. The AWE clients, running on distributed, heterogeneous computing resources, keep checking out workunits from the server queue and dispatching the workunits on the local computing resources. 

AWE uses the Shock data management system to handle input and output data (retrieval, storage, splitting, and merge). AWE uses a RESTful API for communication between AWE components and with outside components such as Shock, the job submitter, and the status monitor.

![awe-diagram](https://raw.githubusercontent.com/MG-RAST/AWE/master/site/images/awe-multi-site.png)


AWE is actively being developed at [github.com/MG-RAST/AWE](http://github.com/MG-RAST/AWE).


Shock is actively being developed at [github.com/MG-RAST/Shock](http://github.com/MG-RAST/Shock).



Documents:
------

A manual for getting started:

https://github.com/MG-RAST/AWE/wiki/Manual-to-Get-Started

A step-by-step manual for running a simple workflow example:

http://www.mcs.anl.gov/~wtang/files/awe-example.pdf

REST API doc:

https://github.com/MG-RAST/AWE/wiki/AWE-APIs

For more documents, check the wiki pages:

https://github.com/MG-RAST/AWE/wiki/_pages

Papers to cite:

W. Tang, J. Wilkening, N. Desai, W. Gerlach, A. Wilke, F. Meyer, "A scalable data analysis platform for metagenomics," in Proc. of IEEE International Conference on Big Data, 2013.[[ieeexplore]](http://ieeexplore.ieee.org/xpl/articleDetails.jsp?arnumber=6691723) [[pdf]](http://www.mcs.anl.gov/papers/P5012-0913_1.pdf)



Google Group:
------

Forum: https://groups.google.com/d/forum/awe-users

Mailing list: awe-users@googlegroups.com  

(bug reports and feature requests are welcomed!)

