This manual shows one workflow example on how to run Shock, AWE server and AWE clients on one ubuntu machine (need to open multiple terminals to simulate different machines). It is tested on ubuntu instances in Magellan cloud. To extend the example to multiple machines, just edit the cfg file to specify correct IP addresses. 

**1. Install AWE, SHOCK, sample applications.**

```shell
wget http://www.mcs.anl.gov/~wtang/files/shockawe_install.sh
source shockawe_install.sh
```

(this installation script can automatically install everything needed on ubuntu linux machine. For other OS, one can follow the comments inside to install the needed software in other ways)

**2. run shock**

```shell
shock-server -conf ~/etc/shock.cfg
```
**3. run awe-server**

```shell
awe-server -conf ~/etc/awes.cfg -dev
```

(with -dev, show queue status printed on screen)

**3. run awe-client**

```shell
awe-client -conf ~/etc/awes.cfg
```

(multiple clients can be started at the same time. also, awe-clients can be started after job is submitted at step 4.)

**4. submit job**

4.1 get sample input

```shell
cd ~  
wget http://www.mcs.anl.gov/~wtang/files/benchmark.fna
```

4.2 get the pipeline template 

```shell
ln -s $GOPATH/src/github.com/MG-RAST/AWE/templates/pipeline_templates/simple_example.template
```

(if you don’t have to make this link, you need to specify the whole path of the template to the job submitting command below)
(this template describes a pipeline with three tasks (in MG-RAST): preprocessing, derepication, and gene prediction (FragGeneScan). These three are used as example because they don’t require installing reference databases (which makes the example simple).

4.3 submit job

```shell
 awe_submit.pl -awe=localhost:8001 -shock=localhost:8000 -upload=benchmark.fna -pipeline=simple_example.template -totalwork=4
```

Note: awe_submit.pl is in $GOPATH/src/github.com/MG-RAST/AWE/scripts. Previous installation script already added that directory into $PATH.
In this example, the sample input file is first uploaded to shock and then the job is submitted with the returned shock id. there are two other use cases of job submission: using an already existed shock id or using an existing instantinated job script. For details see the help of awe_submit.pl by just run awe_submit.pl. without any parameters.

Need more sample input? here is another one in fastq format.

```shell
wget http://www.mcs.anl.gov/~wtang/files/raw.fastq
```

4.4 checking results:

job can be get from REST API

view one specific job:
http://localhost:8001/job/[job_uuid]   [job_uuid] can be got after the job is submitted by awe_submit.pl

view all active jobs:
http://lcoalhost:8001/job?active

the output location (shock id) can be found in the job struct returned by those APIs.

one command line: the job json object can be viewed by:

```shell
curl -X  GET http://localhost:8001/job/ff673324-27da-4337-a9e7-912348d5de6c | python -mjson.tool 
```

 (replace the uuid with the correct one,  python -mjson.tool is for pretty print)


more API usage can be found in the API manual.

view logs:

/mnt/local/awe/logs or /mnt/local/shock/logs (as configured in shock.cfg and awes.cfg)


**5. Modify code and rebuild**

the source code directory is at:
```shell
$GOPATH/src/github.com/MG-RAST/AWE
$GOPATH/src/github.com/MG-RAST/Shock
```

you can make any edit and rebuild by running:
```shell
AWE_BUILD.sh
```
or 
```shell
Shock_BUILD.sh
```

This building script is already downloaded by the installation script to $GOPATH/bin

the Go executables (shock-server, awe-server,  awe-client) will be found in $GOPATH/bin if built successfully.



Author: Wei Tang
