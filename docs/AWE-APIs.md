1. [[Job management APIs|https://github.com/MG-RAST/AWE/wiki/AWE-APIs#wiki-1-job-management-apis]]
2. [[Workunit related API|https://github.com/MG-RAST/AWE/wiki/AWE-APIs#wiki-2-workunit-apis]]
3. [[Client management API|https://github.com/MG-RAST/AWE/wiki/AWE-APIs#wiki-3-client-management-apis]]
4. [[Queue related API|https://github.com/MG-RAST/AWE/wiki/AWE-APIs#wiki-4-queue-management-apis]]
5. [[Logging related API|https://github.com/MG-RAST/AWE/wiki/AWE-APIs#wiki-5-logger-management-apis]]

### 1. Job management APIs

* Job submission

<code>curl ［-H "Datatoken: $TokenString"] -X POST -F upload=@job_script http://\<awe_api_url\>/job</code>

* Job import

<code>curl ［-H "Datatoken: $TokenString"] -X POST -F import=@job_document http://\<awe_api_url\>/job</code>

* Show all jobs 

<code>curl -X GET http://\<awe_api_url\>/job</code>
or
<code>curl -X GET http://\<awe_api_url\>/job?limit=\<INT\>&offset=\<INT\>&order=\<job_field\>&direction=\<asc|desc\></code>

This API returns a page of job objects and total counts. Use limit & offset to control paginated view (by default limit=25, offset=0), to show all jobs, you can set limit=total_counts and offset = 0. By default the jobs will be sorted by last updated time (desc). You can change the sorting criteria by setting  order=\<job_field\>&direction=\<asc|desc\>.

* Show all active jobs in the queue

<code>curl -X GET http://\<awe_api_url\>/job?active[&limit=25&offset=0&order=updatetime&direction=desc]</code>

* Show all suspended jobs in the queue

<code>curl -X GET http://\<awe_api_url\>/job?suspend[&limit=25&offset=0&order=updatetime&direction=desc]</code>

* Show all registered jobs in the queue (active + suspend, excluding jobs in mongodb only)

<code>curl -X GET http://\<awe_api_url\>/job?registered[&limit=25&offset=0&order=updatetime&direction=desc]</code>

* Show one job with specific job id

<code>curl -X GET http://\<awe_api_url\>/job/\<job_id\></code>

* Show one job report with specific job id

<code>curl -X GET http://\<awe_api_url\>/job/\<job_id\>?report</code>

* Query jobs by fields

<code>curl -X GET http://\<awe_api_url\>/job?query&state=\<in-progress|completed\>&info.project=xxx&info.user=xxx&...[?limit=25&offset=0&order=updatetime&direction=desc&distinct=xxx&date_start=2000-01-01&date_end=2010-01-01]</code>

* Suspend an in-progress job

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?suspend</code>

* Resume a suspended job

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?resume</code>
		
* Resume all the suspended jobs in the queue	

<code>curl -X PUT http://\<awe_api_url\>/job?resumeall</code>

* Recover a job from mongodb missing from queue

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?recover</code>

* Recompute a job from task i, the successive / downstream tasks of i will all be recomputed

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?recompute=\<int\></code>

* Resubmit a job, re-start from the beginning, all tasks will be computed

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?resubmit</code>

* Change the clientgroup attribute of the job

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?clientgroup=\<new_cliengroup\></code>

* Change the pipeline attribute of the job

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?pipeline=\<new_pipeline\></code>

* Change the priority attribute of the job

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?priority=\<new_priority\></code>

* Change the expiration attribute of the job, does not get deleted until completed

<code>curl -X PUT http://\<awe_api_url\>/job/\<job_id\>?expiration=\<new_expiration\></code>

* Set job state as deleted, 'full' option deletes job from mongodb and filesystem

<code>curl -X DELETE http://\<awe_api_url\>/job/\<job_id\></code>

<code>curl -X DELETE http://\<awe_api_url\>/job/\<job_id\>?full</code>

* Delete all suspended jobs

<code>curl -X DELETE http://\<awe_api_url\>/job?suspend</code>

<code>curl -X DELETE http://\<awe_api_url\>/job?suspend&full</code>

* Delete all zombie jobs (“in-progress” jobs in mongodb but not in the queue)
			
<code>curl -X DELETE http://\<awe_api_url\>/job?zombie</code>

<code>curl -X DELETE http://\<awe_api_url\>/job?zombie&full</code>


### 2. Workunit APIs

* view a workunit (Highlighted ones are called only by awe-clients)

<code>curl -X GET http://\<awe_api_url\>/work/\<work_id\></code>

* client request data token for the specific workunit

<code>curl -X GET http://\<awe_api_url\>/work/\<work_id\>?datatoken&client=\<client_id\></code>

* client checkout workunit

<code>curl -X GET http://\<awe_api_url\>/work?client=\<client_id\></code>

* client update workunit status (with optional logs/reports in files)

<code>curl -X GET [-F perf=@perf_log] [-F notes=notes_file] http://\<awe_api_url\>/work/\<work_id\>?status=\<new_status\>&client=\<client_id\>&report</code>

* view awe-client error log related to the failed workunit (need client side config: [Client] print_app_msg=True):

<code>curl -X GET http://\<awe_api_url\>/work/\<work_id\>?report=worknotes</code>

* view stdout of one workunit (need client side config: [Client] print_app_msg=True)

<code>curl -X GET http://\<awe_api_url\>/work/\<work_id\>?report=stdout</code>

* view stderr of one workunit (need client side config: [Client] print_app_msg=True)

<code>curl -X GET http://\<awe_api_url\>/work/\<work_id\>?report=stderr</code>


### 3. Client management APIs:

* Register a new client

<code>curl -X POST -F profile=@clientprofile http://\<awe_api_url\>/client</code>

* View all clients

<code>curl -X GET http://\<awe_api_url\>/client</code>

* View all busy clients

<code>curl -X GET http://\<awe_api_url\>/client?busy</code>

* View all clients in clientgroup

<code>curl -X GET http://\<awe_api_url\>/client?group=\<name\></code>

* View all clients by status

<code>curl -X GET http://\<awe_api_url\>/client?status=\<client status\></code>

* View all clients with a given app

<code>curl -X GET http://\<awe_api_url\>/client?app=\<app in list\></code>

* View a client

<code>curl -X GET http://\<awe_api_url\>/client/\<client_id\></code>

* Client sends heartbeat to server

<code>curl -X GET http://\<awe_api_url\>/client/\<client_id\>?heartbeat</code>

* Suspend all the clients

<code>curl -X PUT http://\<awe_api_url\>/client?suspendall</code>

* Resume all suspended clients

<code>curl -X PUT http://\<awe_api_url\>/client?resumeall</code>

* Suspend a client

<code>curl -X PUT http://\<awe_api_url\>/client/\<client_id>?suspend</code>

* Resume a client

<code>curl -X PUT http://\<awe_api_url\>/client/\<client_id\>?resume</code>


### 4. Queue management APIs

* Queue summary, 'json' option for json format

<code>curl -X GET http://\<awe_api_url\>/queue</code>

<code>curl -X GET http://\<awe_api_url\>/queue?json</code>

* Job queue details, requires admin authorization

<code>curl -X GET http://\<awe_api_url\>/queue?job</code>

* Task queue details, requires admin authorization

<code>curl -X GET http://\<awe_api_url\>/queue?task</code>

* Workunit queue details, requires admin authorization

<code>curl -X GET http://\<awe_api_url\>/queue?work</code>

* Client queue details, requires admin authorization

<code>curl -X GET http://\<awe_api_url\>/queue?client</code>

* View running jobs for given clientgroup, requires clientgroup authorization

<code>curl -X GET http://\<awe_api_url\>/queue?clientgroup=\<group name\></code>

* Suspend queue, requires admin authorization

<code>curl -X PUT http://\<awe_api_url\>/queue?suspend</code>

* Resume queue, requires admin authorization

<code>curl -X PUT http://\<awe_api_url\>/queue?resume</code>


### 5. Logger management APIs

* Event code descriptions

<code>curl -X GET http://\<awe_api_url\>/logger?event</code>

* View debug logging level

<code>curl -X GET http://\<awe_api_url\>/logger?debug</code>

* Set debug logging level

<code>curl -X PUT http://\<awe_api_url\>/logger?debug=[0|1|2|3]</code>
