
run awe-server with this option to activate memory profiling
```bash
--memprofile=/mnt/awe/logs/memprofile.prof
```

visualize profile:
```bash
docker exec -ti awe-server-1 ash

cd /mnt/awe/log/

go tool pprof /go/bin/awe-server ./memprofile.prof 

apk add graphviz
go tool pprof -pdf /go/bin/awe-server ./memprofile.prof  > memprofile.pdf
go tool pprof -svg /go/bin/awe-server ./memprofile.prof  > memprofile.svg


scp bio-worker1:/media/ephemeral/awe-server-1/logs/*.pdf .
```