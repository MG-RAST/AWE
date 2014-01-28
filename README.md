![AWE](https://raw.github.com/wtangiit/AWE/master/site/images/awe-lg.png)
=====

About:
------

AWE is a workflow engine that manages and executes scientific computing workflows or pipelines. 


AWE is designed as a distributed system that contains a centralized server and multiple distributed clients. The server receives job submissions and parses jobs into tasks, splits tasks into workunits, and manages workunits in a queue. The AWE clients, running on distributed, heterogeneous computing resources, keep checking out workunits from the server queue and dispatching the workunits on the local computing resources. 


AWE uses the Shock data management system to handle input and output data (retrieval, storage, splitting, and merge). AWE uses a RESTful API for communication between AWE components and with outside components such as Shock, the job submitter, and the status monitor.


AWE is actively being developed at [github.com/MG-RAST/AWE](http://github.com/MG-RAST/AWE).


Shock is actively being developed at [github.com/MG-RAST/Shock](http://github.com/MG-RAST/Shock).



Manual:
------

A detailed manual and API doc of AWE is available at:

http://www.mcs.anl.gov/~wtang/files/awe-manual.pdf

A manual for running a workflow example:

http://www.mcs.anl.gov/~wtang/files/awe-example.pdf

Paper to cite:

[Wei Tang; Wilkening, J.; Desai, N.; Gerlach, W.; Wilke, A.; Meyer, F., "A scalable data analysis platform for metagenomics," Big Data, 2013 IEEE International Conference on, pp.21,26, 6-9 Oct. 2013]
(http://ieeexplore.ieee.org/xpl/articleDetails.jsp?arnumber=6691723&searchWithin%3Dscalable%26sortType%3Dasc_p_Sequence%26filter%3DAND%28p_IS_Number%3A6690588%29)

