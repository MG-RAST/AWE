If a globus token is provided at AWE job submission, awe-clients can use that token to talk to Shock for accessing private data, and the application scripts dispatched at awe-clients can use that token to access other kbase services if the scripts can retrieve the token from an environmental variable named KB_AUTH_TOKEN.

Following is the usage:

1) Submit job with token:

The API for submitting a job with a token is:

<code>curl -H "Datatoken: $TokenString" -X POST -F upload=@job_script http://\<awe_api_url\>/job</code>

Following script serves as an example how to submit data to shock and jobs to awe using APIs. 

https://github.com/MG-RAST/AWE/blob/master/utils/awe_submit.pl

an option with -token="your_token_string" will pass the token with the submission.

2) Specify in the job template to tell which task needs to use token for accessing other kbase services.

https://github.com/MG-RAST/AWE/blob/master/templates/pipeline_templates/simple_example.template

line 18 tells, the task "0" cmd "preprocess" need to retrieve token in the environmental variable named "KB_AUTH_TOKEN".

3) you can use other name than "KB_AUTH_TOKEN", but make sure the script that need to use the token can retrieve environmental variable with the consistent name with what you specified in the workflow document.

Note:

1) For accessing Shock only, nothing needs to be added in the job template and command script, if the job is submitted with the token, the token will be used automatically by awe-client

2) the token will not be shown in the job description json struct, it is stored at AWE server mongodb after job submission, and will be passed from awe-server to awe-client via http header. If the environmental variable will be set, it will be cleaned after the command script finishes.


