### JobPerf

| Attribute     | Type     | Json string |  Notes    |
| ------------- | -------- | ----------- | ------  |
| **Id**       | string | job id |  |  
| **Queued**      | int64 | queued | unix time stamp when job created |
| **Start**      | int64 | start | unix time stamp when job started (first workunit checked out) |
| **End**      | int64 | end | unix time stamp when job completed (last workunit done) |
| **Resp**      | int64 | resp | job response time (in Second) (end - queued) |
| **Ptasks**      | map[string]*TaskPerf | task_stats | detailed per task perf stats, map key is task id |
| **Pworks**      | map[string]*WorkPerf | work_stats | detailed per workunit perf stats, map key is workunit id |

### TaskPerf

| Attribute     | Type     | Json string |  Notes    |
| ------------- | -------- | ----------- | ------  |
| **Queued**      | int64 | queued | unix time stamp when task becomes ready |
| **Start**      | int64 | start | unix time stamp when task started (first workunit checked out) |
| **End**      | int64 | end | unix time stamp when task completed (last workunit done) |
| **Resp**      | int64 | resp | task response time (in Second) (end - queued) |
| **InFileSizes**  | []int64 | size_infile | an array of sizes of each input file (in Byte) |
| **OutFileSizes**  | []int64 | out_infile | an array of sizes of each out file (in Byte) |

### WorkPerf

| Attribute     | Type     | Json string |  Notes    |
| ------------- | -------- | ----------- | ------  |
| **Queued**   |  int64 | queued | unix time stamp when workunit parsed and queued |
| **Done**      | int64 | end | unix time stamp when workunit completed and returned at server )|
| **Resp**      | int64 | resp | workunit response time (in Second) (done - queued) |
| **Checkout**      | int64 | checkout | unix time stamp when workunit checked out at client|
| **Deliver**      | int64 | deliver | unix time stamp when workunit done at client|
| **ClientResp**      | int64 | clientresp | workunit response time at client (in Second)  (deliver - checkout) |
| **DataIn**      | int64 | time_data_in | time in second for input data move-in at client |
| **DataOut**      | int64 | time_data_out | time in second for output data move-out at client  |
| **Runtime**      | int64 | runtime | walltime in second for computation |
| **MaxMemUsage**      | uint64 | max_mem_usage | maximum memory usage during workunit execution (in Byte) |
| **ClientId**      | string |  client_id | the id of the client where workunit is processed |
| **PreDataSize**  | []int64 | size_predata | size in Byte of prerequisite data (e.g. ref db) moved over network |
| **InFileSizes**  | []int64 | size_infile | size in Byte of total input data moved over network  |
| **OutFileSizes**  | []int64 | out_infile | size in Byte of total output data moved over network |