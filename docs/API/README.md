
## 1. Job management APIs

* Job submission

  `curl ［-H "Datatoken: $TokenString"] -X POST -F upload=@job_script http://<awe_api_url>/job`

* Job import

  `curl ［-H "Datatoken: $TokenString"] -X POST -F import=@job_document http://<awe_api_url>/job`

* Show all jobs 

  `curl -X GET http://<awe_api_url>/job`
or
  `curl -X GET http://<awe_api_url>/job?limit=<INT>&offset=<INT>&order=<job_field>&direction=<asc|desc>`

  This API returns a page of job objects and total counts. Use limit & offset to control paginated view (by default limit=25, offset=0), to show all jobs, you can set limit=total_counts and offset = 0. By default the jobs will be sorted by last updated time (desc). You can change the sorting criteria by setting  order=<job_field>&direction=<asc|desc>.

* Show all active jobs in the queue

  `curl -X GET http://<awe_api_url>/job?active[&limit=25&offset=0&order=updatetime&direction=desc]`

* Show all suspended jobs in the queue

  `curl -X GET http://<awe_api_url>/job?suspend[&limit=25&offset=0&order=updatetime&direction=desc]`

* Show all registered jobs in the queue (active + suspend, excluding jobs in mongodb only)

  `curl -X GET http://<awe_api_url>/job?registered[&limit=25&offset=0&order=updatetime&direction=desc]`

* Show one job with specific job id

  `curl -X GET http://<awe_api_url>/job/<job_id>`

* Show one job report with specific job id

  `curl -X GET http://<awe_api_url>/job/<job_id>?report`

* Query jobs by fields

  `curl -X GET http://<awe_api_url>/job?query&state=<in-progress|completed>&info.project=xxx&info.user=xxx&...[?limit=25&offset=0&order=updatetime&direction=desc&distinct=xxx&date_start=2000-01-01&date_end=2010-01-01]`

* Suspend an in-progress job

  `curl -X PUT http://<awe_api_url>/job/<job_id>?suspend`

* Resume a suspended job

  `curl -X PUT http://<awe_api_url>/job/<job_id>?resume`
		
* Resume all the suspended jobs in the queue	

  `curl -X PUT http://<awe_api_url>/job?resumeall`

* Recover a job from mongodb missing from queue

  `curl -X PUT http://<awe_api_url>/job/<job_id>?recover`

* Recompute a job from task i, the successive / downstream tasks of i will all be recomputed

  `curl -X PUT http://<awe_api_url>/job/<job_id>?recompute=<int>`

* Resubmit a job, re-start from the beginning, all tasks will be computed

  `curl -X PUT http://<awe_api_url>/job/<job_id>?resubmit`

* Change the clientgroup attribute of the job

  `curl -X PUT http://<awe_api_url>/job/<job_id>?clientgroup=<new_cliengroup>`

* Change the pipeline attribute of the job

  `curl -X PUT http://<awe_api_url>/job/<job_id>?pipeline=<new_pipeline>`

* Change the priority attribute of the job

  `curl -X PUT http://<awe_api_url>/job/<job_id>?priority=<new_priority>`

* Change the expiration attribute of the job, does not get deleted until completed

  `curl -X PUT http://<awe_api_url>/job/<job_id>?expiration=<new_expiration>`

* Set job state as deleted, 'full' option deletes job from mongodb and filesystem

  `curl -X DELETE http://<awe_api_url>/job/<job_id>`

  `curl -X DELETE http://<awe_api_url>/job/<job_id>?full`

* Delete all suspended jobs

  `curl -X DELETE http://<awe_api_url>/job?suspend`

  `curl -X DELETE http://<awe_api_url>/job?suspend&full`

* Delete all zombie jobs (“in-progress” jobs in mongodb but not in the queue)
			
  `curl -X DELETE http://<awe_api_url>/job?zombie`

  `curl -X DELETE http://<awe_api_url>/job?zombie&full`


## 2. Workunit APIs

* view a workunit (Highlighted ones are called only by awe-clients)

  `curl -X GET http://<awe_api_url>/work/<work_id>`

* client request data token for the specific workunit

  `curl -X GET http://<awe_api_url>/work/<work_id>?datatoken&client=<client_id>`

* client checkout workunit

  `curl -X GET http://<awe_api_url>/work?client=<client_id>`

* client update workunit status (with optional logs/reports in files)

  `curl -X GET [-F perf=@perf_log] [-F notes=notes_file] http://<awe_api_url>/work/<work_id>?status=<new_status>&client=<client_id>&report`

* view awe-client error log related to the failed workunit (need client side config: [Client] print_app_msg=True):

  `curl -X GET http://<awe_api_url>/work/<work_id>?report=worknotes`

* view stdout of one workunit (need client side config: [Client] print_app_msg=True)

  `curl -X GET http://<awe_api_url>/work/<work_id>?report=stdout`

* view stderr of one workunit (need client side config: [Client] print_app_msg=True)

  `curl -X GET http://<awe_api_url>/work/<work_id>?report=stderr`


### 3. Client management APIs:

* Register a new client

  `curl -X POST -F profile=@clientprofile http://<awe_api_url>/client`

* View all clients

  `curl -X GET http://<awe_api_url>/client`

* View all busy clients

  `curl -X GET http://<awe_api_url>/client?busy`

* View all clients in clientgroup

  `curl -X GET http://<awe_api_url>/client?group=<name>`

* View all clients by status

  `curl -X GET http://<awe_api_url>/client?status=<client status>`

* View all clients with a given app

  `curl -X GET http://<awe_api_url>/client?app=<app in list>`

* View a client

  `curl -X GET http://<awe_api_url>/client/<client_id>`

* Client sends heartbeat to server

  `curl -X GET http://<awe_api_url>/client/<client_id>?heartbeat`

* Suspend all the clients

  `curl -X PUT http://<awe_api_url>/client?suspendall`

* Resume all suspended clients

  `curl -X PUT http://<awe_api_url>/client?resumeall`

* Suspend a client

  `curl -X PUT http://<awe_api_url>/client/<client_id>?suspend`

* Resume a client

  `curl -X PUT http://<awe_api_url>/client/<client_id>?resume`


## 4. Queue management APIs

* Queue summary, 'json' option for json format

  `curl -X GET http://<awe_api_url>/queue`

  `curl -X GET http://<awe_api_url>/queue?json`

* Job queue details, requires admin authorization

  `curl -X GET http://<awe_api_url>/queue?job`

* Task queue details, requires admin authorization

  `curl -X GET http://<awe_api_url>/queue?task`

* Workunit queue details, requires admin authorization

  `curl -X GET http://<awe_api_url>/queue?work`

* Client queue details, requires admin authorization

  `curl -X GET http://<awe_api_url>/queue?client`

* View running jobs for given clientgroup, requires clientgroup authorization

  `curl -X GET http://<awe_api_url>/queue?clientgroup=<group name>`

* Suspend queue, requires admin authorization

  `curl -X PUT http://<awe_api_url>/queue?suspend`

* Resume queue, requires admin authorization

  `curl -X PUT http://<awe_api_url>/queue?resume`


## 5. Logger management APIs

* Event code descriptions

  `curl -X GET http://<awe_api_url>/logger?event`

* View debug logging level

  `curl -X GET http://<awe_api_url>/logger?debug`

* Set debug logging level

  `curl -X PUT http://<awe_api_url>/logger?debug=[0|1|2|3]`
