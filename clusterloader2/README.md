# Cluster Loader

This is mod for [CL2](http://github.com/kubernetes/perf-tests/tree/master/clusterloader2) that brings cluster loader for [kueue](http://github.com/kubernetes-sigs/kueue)

Motivation for this tool is to write a adapt kueue specific test scenarios and measurements. Yet it is easier to embed into the kueue as this tool has been distributed as a standalone binary artifact that already contains everything needed inside (batteries included)

## Compile

Presumingly golang already configured on your workstation

```bash
go get
go build
```

> Warning: this is alpha (or even pre-alpha) maturity. Use this on your own risk

# Directory structure

```
clusterloader2
├── go.mod
├── main.go                        # golang entrypoint 
├── measurement
│   └── custom1.go                 # boiler plate for custom measurement
├── reports                        # directory for test reports 
│   └── junit.xml                  # generated test report
└── scenarios                      # default directory to discover test scenarios
    ├── config*.yaml               # test config that will be automatically detected and executed
    ├── cluster-queue.yaml
    ├── job.yaml
    ├── local-queue.yaml
    └── resource-flavor.yaml
```

# TODOs

Initial todos for this tool

- [ ] Add support for other providers (currently only "local")
- [ ] Add prometheus framework
- [ ] Discover cluster configuration (nodes, masters)
- [ ] Add kueue latency metrics

