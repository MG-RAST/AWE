1. [Job](#job)
2. [Task](#task)
3. [Workunit](#workunit)
4. [Client](#client)
5. [Info](#info)
6. [IO](#io)
7. [Command](#command) 

# Job

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | ------ | -------- |
| **Id**       | string | id | e.g. c779a7f7-953d-4079-8388-591ee2065bad | uuid |
| **Jid**      | string | jid | e.g. 12367 | incremental job id  per awe-server domain |
| **Info** | \*Info      |   info |   | job info |
| **Tasks** | []\*Task   |   tasks |   |an array of task struct |
| **State** | string     |   state | **init**: initial state at job submission<br />**queued**:parsed and waiting in queue<br />**in-progress**:at least one workunit is out<br />**completed**: all tasks done<br />**suspend**: paused for failure or manually<br />**deleted**: manually removed||
| **Registered** | bool     |   registered |**true**: in the queue (in memory) <br />**false**: in mongodb only  |  |
| **RemainTasks** | int      |   remaintasks |0 -- len(Tasks)   | number of tasks not done |
| **UpdateTime** | time.Time      |   updatetime |e.g."2014-02-09T15:43:40.574Z"  | timestamp for last state update |
| **Notes** | string      |   notes |e.g."job suspended for xxx reason"  | notes for last state update |
| **LastFailed** | string      |   lastfailed | 99abf2eb-380f-47ec-8d43-11be265b4594_2_0  | the id of the task or workunit which causes the job suspension (more detailed reason can be found in Notes) |

# Task

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | --------- | ----- |
| **Id**       | string | id | e.g. c779a7f7-953d-4079-8388-591ee2065bad_1 | jobid_stage |
| **Info** | \*Info      |   info |   | job info (inherited) |
| **Inputs** |  IOmap (map[string]*IO)     |   inputs |   | list of input files |
| **Outputs** |  IOmap (map[string]*IO)      |  outputs |   | list of output files |
| **Predata** |  IOmap (map[string]*IO)     |  predata |   | list of prerequisite data (e.g. reference dbs) |
| **Cmd**      | \*Command  | cmd |  | cmd definition (cmd name, args, etc) |
| **DependOn** | []string   | dependsOn  |   | the list of parent tasks (predecessor in the DAG) |
| **MaxWorkSize** | int     | maxworksize | e.g. 50 | max input per workunit (in MB) |
| **TotalWork** | int   |  totalwork | e.g. 16   | number of workunits to split |
| **RemainWork** | int   |  remainwork | e.g. 7   | number of workunits that not done |
| **State** | string     |   state | **init**: initial state<br />**queued**:ready and waiting in queue<br />**in-progress**:at least one workunit is out<br />**pending**:not parsed yet for parent task not done<br />**completed**: all workunits done<br />**suspend**: paused for failure or manually<br />  |  |
| **CreatedDate** | time.Time      |   createdate |e.g."2014-02-09T15:43:40.574Z"  | timestamp (“queued”) |
| **StartedDate** | time.Time      |   starteddate |e.g."2014-02-09T15:45:40.574Z"  | timestamp for (“in-progress”) |
| **CompletedDate** | time.Time      |  completeddate |e.g."2014-02-09T15:46:40.574Z"  | timestamp for (“completed”) |
| **ComputeTime** | int    |  computetime  |e.g. 3600  |  aggregated  workunit compute walltime (in second) |

# Workunit

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | ------ | -------- |
| **Id**       | string | id | e.g. c779a7f7-953d-4079-8388-591ee2065bad_1_5 | jobid_stage_rank |
| **Info** | \*Info      |   info |   | job info (inherited) |
| **Inputs** |  IOmap      |   inputs |   | inherited from task |
| **Outputs** |  IOmap      |  outputs |   |  inherited from task |
| **Predata** |  IOmap      |  predata |   | inherited from task |
| **Cmd**      | \*Command  | cmd |  | inherited from task  |
| **Rank** | int   | rank  | 0 or 5  | 0 means it is the only workunit, an integer >0 means it is one of multiple splits |
| **State** | string     |   state |  **queued**:waiting in queue<br />**checkout**:being checked out by some client<br />**completed**:successfully done<br />**suspend**: failed thus suspended |  |
| **Failed** | int      |  e.g. 2  | number of times failed (can result in “suspend” if larger than threshold (e.g. 5) |
| **CheckoutTime** | time.Time      |   checkout_time |e.g."2014-02-09T15:43:40.574Z"  | timestamp being checked out |
| **Client** |  string    |  client |   | id of the client which is processing this workunit |
| **ComputeTime** | int    | computetime  |e.g. 600  |  compute walltime (in second) |

# Client

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | ------ | -------- |
| **Id**       | string | id | e.g. c779a7f7-953d-4079-8388-591ee2065bad | uuid |
| **Name**      | string  | name |  | human readable name |
| **Group** |  string   | group  |   | representing logical “queue’” name, matching job group name |
| **User** | string   |  user |   | owner|
| **Domain** | string     |   domain | e.g. “magellan” | representing physical region of computing resources |
| **InstanceId** | string     |  instance_id | e.g. 000dfa | openstack instance id (optional) |
| **InstanceType** | string     |  instance_type | idp100 | openstack instance type/flavor (optional) |
| **RemainTasks** | int      |   remaintasks |0 -- len(Tasks)   | number of tasks not done |
| **Host** | string      |  string |    | host ip address |
| **CPUs** | int      | cores   | e.g. 8 | number of cores |
| **Apps** | []string      | apps | [“fgs”,”bowtie”] | array, supported commands,  “*” for any app|
| **RegTime** | time.Time      | regtime |  | time stamp the clients first registered |
| **Serve_time** | Serve_time      | serve_time | e.g. 50h16m | time period since first reigstered |
| **Idle_time** | int      | idle_time |  3600 | time (in second) since last time idling |
| **Status** | string      | Status | **active-idle**: up running but idle<br />**active-busy**:processing some workunit(s) <br />**suspend**:not able to get workunits (for failure or manually paused) <br /> | |
| **Total_checkout** | int      | total_checkout |  e.g. 50 | number of workunits checked out by this client|
| **Total_completed** | int      | total_checkout |  e.g. 45 | number of work units completed at this client |
| **Total_failed** | int      | total_failed |  e.g. 5 | number of work units failed at this client |
| **Current_work** | map[string]bool   current_work   | {\<workunit_id\>} |  id list of workunits being processed by this client  |
| **Skip_work** | []string | skip_work    |[\<workunit_id\>] |  id list of workunits failed at this client (skip checking out next time)  |
| **Proxy** | bool     proxy| **True** means this client is a proxy  | |
| **SubClients** | int  | subclients  | e.g. 5 |  number of  subclients registered (valid only when proxy=true |

# Info

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | ------ | -------- |
| **Name**       | string | name | job name  |  |
| **Project**       | string | project | job name  |  |
| **User**       | string | user | user name  |  |
| **Pipeline**       | string | pipeline | pipeline/workflow name  |  |
| **ClientGroups**       | string | clientgroups | a list of client group that can run this job  | group_name_a,group_name_b |
| **SubmitTime** |  time.Time   | submittime  | job submit time |  |
| **StartedTime** | time.Time  |  startedtime | job started time (first workunit started)| |
| **CompletedTime** | time.time  |  completedtime | job completed time |  |
| **Priority** | int     |  priority | priority level | default 1, can be changed to any integer
| **Auth** | bool     |  auth | if true, means the job needs data token to access private shock data | |
| **NoRetry** |  bool    |  noretry | if set to true, the workunit will fail without retry (other wise will retry 3 times before being suspended) | |
| **UserAttr** | map[string]string      | userattr |  user specified job attributes  | |

type PartInfo struct {
	Input         string `bson:"input" json:"input"`
	Index         string `bson:"index" json:"index"`
	TotalIndex    int    `bson:"totalindex" json:"totalindex"`
	MaxPartSizeMB int    `bson:"maxpartsize_mb" json:"maxpartsize_mb"`
	Options       string `bson:"options" json:"-"`
}

# IO

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | ------ | -------- |
| **Name**       | string | name | the name of the io object (file name) |  |
| **Host**       | string | host | shock host  |  |
| **Node**       | string | node | shock node id  |  |
| **Url**       | string | url | downloading url  |  |
| **Size**       | string | size | file size  |  |
| **Origin**       | string | origin | parent task which produces this file as output  |  |
| **Nonzero**       | bool | nonzero | if set true, this file must be nonzero; if zero, report failure |  | 
| **ShockFilename**     | string | shockfilename | if this name is non-empty, use this name other than that in "name" field when uploading file to shock  |  |
| **ShockIndex**       | string | shockindex | if non-empty, create index using this index type after uploading data to shock |  |
| **AttrFile**       | string | attrfile | specify the attribute file path to be uploaded to shock as metadata |  | 

# Command

| Attribute     | Type     | Json string | Values | Notes    |
| ------------- | -------- | ----------- | ------ | -------- |
| **Name**       | string | name | the name of the command (executable) |  |
| **Args**       | string | args | arge list in a string |  |
| **Docerimage** | string | dockerimage | docker image url  |  (optional) |
| **Environs**   | Envs | environ | environmental variables |  (optional) |
| **HasPrivateEnv**   | bool | has_private_env | if set, Environs has non-empty "Private" field |  |
| **Description** | string | description | description of this command  |  |
