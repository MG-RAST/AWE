
# AWE server
```

[Ports]
site-port=<int>             Internal port to run AWE Monitor on (default: 8081)
api-port=<int>              Internal port for API (default: 80)

[External]
site-url=<string>           External URL of AWE monitor, including port (default: "http://localhost:8081")
api-url=<string>            External API URL of AWE server, including port (default: "http://localhost:80")

[SSL]
enable=<bool>                (default: false)
key=<string>                 (default: "")
cert=<string>                (default: "")

[Anonymous]
write=<bool>                 (default: true)
read=<bool>                  (default: true)
delete=<bool>                (default: true)
cg_write=<bool>              (default: false)
cg_read=<bool>               (default: false)
cg_delete=<bool>             (default: false)

[Auth]
basic=<bool>                 (default: false)
globus_token_url=<string>    (default: "")
globus_profile_url=<string>  (default: "")
oauth_urls=<string>          (default: "")
oauth_bearers=<string>       (default: "")

[WebApp]
login_url=<string>           (default: "")
client_auth_required=<bool>  (default: false)
auth_oauthserver=<bool>      (default: false)
auth_url=<string>            (default: "")

[Admin]
users=<string>               (default: "")
email=<string>               (default: "")

[Directories]
site=<string>               the path to the website (default: "/go/src/github.com/MG-RAST/AWE/site")
awf=<string>                 (default: "")
data=<string>               a file path for storing system related data (job script, cached data, etc) (default: "/mnt/data/awe/data")
logs=<string>               a path for storing logs (default: "/mnt/data/awe/logs")

[Paths]
pidfile=<string>             (default: "")

[Mongodb]
hosts=<string>               (default: "localhost")
database=<string>            (default: "AWEDB")
user=<string>                (default: "")
password=<string>            (default: "")
timeout=<int>                (default: 1200)

[Server]
title=<string>               (default: "AWE Server")
coreq_length=<int>          length of checkout request queue (default: 100)
expire_wait=<int>           wait time for expiration reaper in minutes (default: 60)
global_expire=<string>      default number and unit of time after job completion before it expires (default: "")
pipeline_expire=<string>    comma seperated list of pipeline_name=expire_days_unit, overrides global_expire (default: "")
perf_log_workunit=<bool>    collecting performance log per workunit (not working) (default: false)
max_work_failure=<int>      number of times that one workunit fails before the workunit considered suspend (default: 1)
max_client_failure=<int>    number of times that one client consecutively fails running workunits before the client considered suspend (default: 0)
go_max_procs=<int>           (default: 0)
reload=<string>             path or url to awe job data. WARNING this will drop all current jobs (default: "")
recover=<bool>              load unfinished jobs from mongodb on startup (default: false)
recover_max=<int>           max number of jobs to recover, default (0) means recover all (default: 0)

[Docker]
use_docker=<string>         "yes", "no" or "only" (default: "yes")
     yes: allow docker tasks, no: do not allow docker tasks, only: allow only docker tasks; if docker is not installed on the clients, choose "no"
use_app_defs=<string>       "yes", "no" or "only" (default: "no")
     yes: allow app defs, no: do not allow app defs, only: allow only app defs
app_registry_url=<string>   URL for app defintions (default: "https://raw.githubusercontent.com/MG-RAST/Skyport/master/app_definitions/")

[Proxy]
p-site-port=<int>            (default: 8082)
p-api-port=<int>             (default: 8002)

[Other]
errorlength=<int>           amount of App STDERR to save in Job.Error (default: 5000)
dev=<bool>                  dev or demo mode, print some msgs on screen (default: false)
conf=<string>               path to config file (default: "")
logoutput=<string>          log output stream, one of: file, console, both (default: "console")
debuglevel=<int>            debug level: 0-3 (default: 0)
version=<bool>              show version (default: false)
fullhelp=<bool>             show detailed usage without "--"-prefixes (default: false)
help=<bool>                 show usage (default: false)
cpuprofile=<string>         e.g. create cpuprofile.prof (default: "")
memprofile=<string>         e.g. create memprofile.prof (default: "")

```


# AWE worker
```

[Directories]
data=<string>               a file path for storing system related data (job script, cached data, etc) (default: "/mnt/data/awe/data")
logs=<string>               a path for storing logs (default: "/mnt/data/awe/logs")

[Paths]
pidfile=<string>             (default: "")

[Client]
serverurl=<string>          URL of AWE server, including API port (default: "http://localhost:8001")
cwl_tool=<string>           CWL CommandLineTool file (default: "")
cwl_job=<string>            CWL job file (default: "")
group=<string>              name of client group (default: "default")
shockurl=<string>           URL of SHOCK server, including port number (default: "http://localhost:8001")
name=<string>               default determines client name by openstack meta data (default: "default")
hostname=<string>           host name (default: "localhost")
     host name to help finding machines where the clients runs on
host_ip=<string>            ip address (default: "127.0.0.1")
     ip address to help finding machines where the clients runs on
host=<string>               deprecated (default: "")
     deprecated
domain=<string>              (default: "default")
clientgroup_token=<string>   (default: "")
supported_apps=<string>     list of suported apps, comma separated (default: "")
app_path=<string>           the file path of supported app (default: "")

[predata Directory]
predata=<string>            a file path for storing predata, by default same as --data (default: "")

[Client]
workpath=<string>           the root dir for workunit working dirs (default: "/mnt/data/awe/work")
metadata=<string>            (default: "")
     e.g. ec2, openstack...
pre_work_script=<string>     (default: "")
pre_work_script_args=<string>  (default: "")
print_app_msg=<bool>        collect stdout/stderr for apps (default: true)
worker_overlap=<bool>       overlap client side computation and data movement (default: false)
auto_clean_dir=<bool>       delete workunit directory to save space after completion, turn of for debugging (default: true)
cache_enabled=<bool>         (default: false)
no_symlink=<bool>           copy files from predata to work dir, default is to create symlink (default: false)
cwl_runner_args=<string>    arguments to pass (default: "")

[Docker]
use_docker=<string>         "yes", "no" or "only" (default: "yes")
     yes: allow docker tasks, no: do not allow docker tasks, only: allow only docker tasks; if docker is not installed on the clients, choose "no"
docker_binary=<string>      docker binary to use, default is the docker API (API recommended) (default: "API")
mem_check_interval_seconds=<int> memory check interval in seconds (kernel needs to support that) (default: 0)
     0 seconds means disabled
cgroup_memory_docker_dir=<string> path to cgroup directory for docker (default: "/sys/fs/cgroup/memory/docker/[ID]/memory.stat")
docker_socket=<string>      docker socket path (default: "unix:///var/run/docker.sock")
docker_workpath=<string>    work dir in docker container started by client (default: "/workdir/")
docker_data=<string>        predata dir in docker container started by client (default: "/db/")
image_url=<string>          url of shock server hosting docker images (default: "http://shock-internal.metagenomics.anl.gov")

[Proxy]
p-site-port=<int>            (default: 8082)
p-api-port=<int>             (default: 8002)

[Other]
errorlength=<int>           amount of App STDERR to save in Job.Error (default: 5000)
dev=<bool>                  dev or demo mode, print some msgs on screen (default: false)
conf=<string>               path to config file (default: "")
logoutput=<string>          log output stream, one of: file, console, both (default: "console")
debuglevel=<int>            debug level: 0-3 (default: 0)
version=<bool>              show version (default: false)
fullhelp=<bool>             show detailed usage without "--"-prefixes (default: false)
help=<bool>                 show usage (default: false)
cpuprofile=<string>         e.g. create cpuprofile.prof (default: "")
memprofile=<string>         e.g. create memprofile.prof (default: "")

```


# AWE submitter

The AWE submitter can be used to submit CWL workflows to the AWE server

```

[Client]
serverurl=<string>          URL of AWE server, including API port (default: "http://localhost:8001")
cwl_tool=<string>           CWL CommandLineTool file (default: "")
cwl_job=<string>            CWL job file (default: "")
group=<string>              name of client group (default: "default")
shockurl=<string>           URL of SHOCK server, including port number (default: "http://localhost:8001")
outdir=<string>             location of output files (default: "")
quiet=<bool>                useless flag for CWL compliance test (default: false)
pack=<bool>                 invoke cwl-runner --pack first (default: false)
wait=<bool>                 wait fopr job completion (default: false)
output=<string>             cwl output file (default: "")
download_files=<bool>       download output files from shock (default: false)
shock_auth=<string>         format: "<bearer> <token>" (default: "")
awe_auth=<string>           format: "<bearer> <token>" (default: "")
job_name=<string>           name of job, default is filename (default: "")
upload_input=<bool>         upload job input files into shock and return new job input structure (default: false)

[Other]
debuglevel=<int>            debug level: 0-3 (default: 0)
version=<bool>              show version (default: false)
fullhelp=<bool>             show detailed usage without "--"-prefixes (default: false)
help=<bool>                 show usage (default: false)
cpuprofile=<string>         e.g. create cpuprofile.prof (default: "")
memprofile=<string>         e.g. create memprofile.prof (default: "")

```
