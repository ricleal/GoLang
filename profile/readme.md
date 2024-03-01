# pprof experiments

# Run the pprof in the benchmark

```sh
go test -bench=. ./profile/hw/...

# with cpu and mem profile
go test -cpuprofile /tmp/cpu.prof -memprofile /tmp/mem.prof -bench=. ./profile/hw/...

# view the profile
go tool pprof /tmp/cpu.prof 
# (pprof) web
# (pprof) 
```

## Run the pprof in the server


```sh
go tool pprof http://localhost:6060/debug/pprof/heap
# (pprof) web

```

