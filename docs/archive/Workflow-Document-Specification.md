## Overview and Examples:

We can specify a workflow document, which when submitted to awe becomes a job document. The workflow and job document are both json formatted. The workflow documents gets turned into a job document in the server. The job contains additional information and the content of the job document changes as the different tasks in the job complete.

Following is a sample workflow document (job template):

https://github.com/MG-RAST/AWE/blob/master/templates/pipeline_templates/simple_example.template

Following is a sample job document (instantiated job json struct):

http://140.221.84.148:8000/job/fdcafcec-f66c-4d37-be5c-8bfbf7cd268f 

the link is from a testing awe-server, if that server is not serving, go to following link for the same content:

http://www.mcs.anl.gov/~wtang/files/sample_awe_job.json

To submit a job, the #hashtags inside the workflow template will need to be replaced by words with	real information. The command line job submitter can do this. One example job submitter is available at: 

https://github.com/MG-RAST/AWE/blob/master/utils/awe_submit.pl

Users can write their one job submitter based on that example.

## Detailed descriptions

### workflow and jobs

Both documents contain at the top level an info field and a tasks field. The info field is a simple structure while the tasks field is more complex. The tasks field evaluates to an array of task objects.

The workflow document looks like this. Tasks will be covered further down, so for this illustration the tasks array is intentionally left empty.

	{
    	"info": {
        	"pipeline": "mgrast-prod",
        	"name": "mgrast_prodtest_1",
        	"project": "kb_test",
        	"user": "default",
        	"clientgroups":""
    	},
    	"tasks": [ intentionally left empty for this illustration ],
	}

The job document looks like this. The bold fields are those that are new, that is to say those that were not in the workflow document. We will see later that the fields in the tasks list also differ between the workflow and the job documents.

	{
	"data": {
    	"id": "9095be9e-12ad-48dd-ac46-c19dbad8a242",
    	"info": {
        	"pipeline": "mgrast-prod",
        	"name": "mgrast_prodtest_1",
        	"project": "kb_test",
        	"user": "default",
        	"clientgroups": "",
        	"submittime": "2013-12-17T09:08:14.596-05:00"
    	},
    	"jid": "10077",
    	"notes": "",
    	"remaintasks": 4,
    	"state": "in-progress",
    	"tasks": [ intentionally left empty for this illustration ],
    	"updatetime": "2013-12-17T09:08:35.729-05:00"
	},
	"error": null,
	"status": 200
	}

The clientgroups field is a string. Because it is plural, we assume multiple client groups can be listed. Therefore, we assume a comma separated list represented as a string.

In the awe client config block, there is a field named group.  You can assign a value to that field and clients that start up using that config file broadcast to the server that it is a member of group X. Then, when you construct a workflow document, you can specify to the server to run tasks on clients of group X. An example could be the case where you had a set of clients in the “test” group and a set of clients in the “production” group.

### task object

The task object has the following fields defined in a workflow document.

Lets look at the cmd field first. Here are the command objects in a sequential workflow where each command takes the output of the previous command as input. Each of these cmd objects is contained in a different task object.

	
    "cmd": {
            "args": "-input=@schloss.all.fasta -output=13.awe_rna_blat-job.prep.fna",
            "description": "preprocess",
            "name": "awe_preprocess.pl"
            },                 

    cmd": {
          "name": "awe_rna_search.pl",
          "args": "-input=@13.awe_rna_blat-job.prep.fna -output=13.awe_rna_blat-job.search.rna.fna -rna_nr=md5nr.clust",
          "description": "rna detection"
    },
       
The first thing that one notices is that the named output file of the first cmd is the named input file to the second cmd. Although, the named input file to the second command is preceded with an @ symbol. Lets look a bit further at the task specifications.


        cmd": {
            "name": "awe_rna_search.pl",
            "args": "-input=@13.awe_rna_blat-job.prep.fna -output=13.awe_rna_blat-job.search.rna.fna -rna_nr=md5nr.clust",
            "description": "rna detection"
       },
       "inputs": {
             "13.awe_rna_blat-job.prep.fna": {
             "host": "http://localhost:7044",
             "origin": "0"
       }

You will notice that the file names match between the two objects. You could also infer that the @ symbol indicates that this filename can be looked up in the inputs object, and also that it can be found in shock.

The origin field of the inputs object indicates the taskid if the command that that produces the input.

### Splitting Tasks into Work Units

To split a task into smaller work units that can run on different awe clients in parallel, the task input is split into smaller inputs. There are two ways to define the splitting: a) by specifying the number of splits, and b) by specifying the maximum size of each split.

The number of splits and the maximum size of each split is specified in the task section of a workflow document. For example:

            	"totalwork": 4,
            	"maxworksize": 50

Setting totalwork equal to 4 means to split the task into 4 workuntis and setting maxworksize equal to 50 means the max input size will be made no larger than 50MB. The maxworksize setting has higher priority than totalwork. This means the final number of splits could be dynamic (calculated by totalsize / maxworksize + 1) and exceed the totalwork setting of 4.

Another use case is that if we don't care about number of splits, we can rely on the maxworksize setting. (e.g. for the clustering stage, we just want to make sure each split is less than 250M, for example:

            	"totalwork": 1,
            	"maxworksize": 250 

Currently splitting of FASTA, FASTQ and BAM format is supported and the format is auto-detected.
