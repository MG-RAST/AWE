AWE server and client (i.e. worker) can be configured via configuration files and the command line. The --help of each binary list all configuration possibilities and their defaults. Example configuration files can be found here:

Sample server cfg template:

```shell
https://github.com/MG-RAST/AWE/blob/master/templates/awes.cfg.template
```

Sample client cfg template:

```shell
https://github.com/MG-RAST/AWE/blob/master/templates/awec.cfg.template
```


To get the latest version of the configuration possibilities please use --fullhelp for each binary.

### AWE server configuration

```shell
> awe-server --fullhelp

[Ports]
site-port=<int>             Internal port to run AWE Monitor on (default: 8081)
api-port=<int>              Internal port for API (default: 8001)

[External]
site-url=<string>           External URL of AWE monitor, including port (default: "")
api-url=<string>            External API URL of AWE server, including port (default: "")

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
basic=<bool>                 (default: true)
globus_token_url=<string>    (default: "")
globus_profile_url=<string>  (default: "")
mgrast_oauth_url=<string>    (default: "")
client_auth_required=<bool>  (default: false)

[Admin]
users=<string>               (default: "")
email=<string>               (default: "")

[Directories]
site=<string>               the path to the website (default: "/gopath//src/github.com/MG-RAST/AWE/site")
awf=<string>                 (default: "")
data=<string>               a file path for store some system related data (job script, cached data, etc) (default: "/mnt/data/awe/data")
logs=<string>               a path for storing logs (default: "/mnt/data/awe/logs")

[Paths]
pidfile=<string>             (default: "")

[Mongodb]
hosts=<string>               (default: "")
database=<string>            (default: "AWEDB")
user=<string>                (default: "")
password=<string>            (default: "")

[Server]
title=<string>               (default: "AWE Server")
perf_log_workunit=<bool>    collecting performance log per workunit (default: true)
max_work_failure=<int>      number of times that one workunit fails before the workunit considered suspend (default: 3)
max_client_failure=<int>    number of times that one client consecutively fails running workunits before the client considered suspend (default: 5)
go_max_procs=<int>           (default: 0)
reload=<string>             path or url to awe job data. WARNING this will drop all current jobs (default: "")
recover=<bool>              path to awe job data (default: false)

[Docker]
use_docker=<string>         "yes", "no" or "only" (default: "yes")
     yes: allow docker tasks, no: do not allow docker tasks, only: allow only docker tasks ; if docker is not installed on the clients, choose "no"
use_app_defs=<string>       "yes", "no" or "only" (default: "no")
     yes: allow app defs, no: do not allow app defs, only: allow only app defs
app_registry_url=<string>   URL for app defintions (default: "https://raw.githubusercontent.com/MG-RAST/Skyport/master/apps.json")

[Proxy]
p-site-port=<int>            (default: 8082)
p-api-port=<int>             (default: 8002)

[Other]
debuglevel=<int>            debug level: 0-3 (default: 0)
dev=<bool>                  dev or demo mode, print some msgs on screen (default: false)
conf=<string>               path to config file (default: "")
version=<bool>              show version (default: false)
fullhelp=<bool>             show detailed usage without "--"-prefixes (default: false)
help=<bool>                 show usage (default: false)
```
### AWE client configuration:
```shell
> awe-client --fullhelp

[Directories]
data=<string>               a file path for store some system related data (job script, cached data, etc) (default: "/mnt/data/awe/data")
logs=<string>               a path for storing logs (default: "/mnt/data/awe/logs")

[Paths]
pidfile=<string>             (default: "")

[Client]
serverurl=<string>          URL of AWE server, including API port (default: "http://localhost:8001")
group=<string>              name of client group (default: "default")
name=<string>               default determines client name by openstack meta data (default: "default")
domain=<string>              (default: "default")
clientgroup_token=<string>   (default: "")
supported_apps=<string>     list of suported apps, comma separated (default: "")
app_path=<string>           the file path of supported app (default: "")
workpath=<string>           the root dir for workunit working dirs (default: "/mnt/data/awe/work")
openstack_metadata_url=<string>  (default: "http://169.254.169.254/2009-04-04/meta-data")
pre_work_script=<string>     (default: "")
pre_work_script_args=<string>  (default: "")
print_app_msg=<bool>        collect stdout/stderr for apps (default: true)
worker_overlap=<bool>       overlap client side computation and data movement (default: false)
auto_clean_dir=<bool>       delete workunit directory to save space after completion, turn of for debugging (default: true)
cache_enabled=<bool>         (default: false)

[Docker]
use_docker=<string>         "yes", "no" or "only" (default: "yes")
     yes: allow docker tasks, no: do not allow docker tasks, only: allow only docker tasks ; if docker is not installed on the clients, choose "no"
docker_binary=<string>      docker binary to use, default is the docker API (API recommended) (default: "API")
mem_check_interval_seconds=<int> memory check interval in seconds (kernel needs to support that) (default: -1)
cgroup_memory_docker_dir=<string> path to cgroup directory for docker (default: "/sys/fs/cgroup/memory/docker/")

[Proxy]
p-site-port=<int>            (default: 8082)
p-api-port=<int>             (default: 8002)

[Other]
debuglevel=<int>            debug level: 0-3 (default: 0)
dev=<bool>                  dev or demo mode, print some msgs on screen (default: false)
conf=<string>               path to config file (default: "")
version=<bool>              show version (default: false)
fullhelp=<bool>             show detailed usage without "--"-prefixes (default: false)
help=<bool>                 show usage (default: false)
```