## 1. Build AWE workflow and submit jobs

Jobs are submitted to AWE in form of workflow documents (in JSON format). Those can be generated in various ways, e.g. with Perl or Python scripts or using workflow templates. Workflow templates look just like the actual workflow but contain placeholders for input data or parameters. To create a workflow that can be submitted to AWE these placeholders have to replaced with actual values. 

### 1.1 Build workflow template

Here is a sample workflow template:

https://github.com/MG-RAST/AWE/blob/master/templates/pipeline_templates/simple_example.template

A detailed description of workflow document specification can be found here:

https://github.com/MG-RAST/AWE/wiki/Workflow-Document-Specification

### 1.2 Submit a job

Jobs (i.e. workflow documents) are submitted via REST API (as POST requests). For user convenience, we provide a Perl script to wrap the API. The script is available at:

https://github.com/MG-RAST/AWE/blob/master/utils/awe_submit.pl

Usage:

```shell
awe_submit.pl ­-awe=localhost:8001 ­-shock=localhost:8000 ­-upload=benchmark.fna ­-pipeline=simple_example.template ­-token="your_kbase_oauth_token"
```

Detailed use case description can be found inside the script.

