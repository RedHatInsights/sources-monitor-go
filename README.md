# Sources Monitor
##### (but in Go)

This tiny little application handles running the availability checks on a pre-determined schedule on OCP.

The `deploy/clowdapp.yaml` file defines the schedule - but as of right now it checks:
|status|schedule|
|---|---|
|Available|:00 of every hour|
|Unavailable|:15/:45 every hour|

The application itself is really simple, it does this order of operations:
1. `GET /internal/v2.0/sources` -- get a list of sources w/ tenants from the internal API
2. Loop through all of them (paginated)
   1. If: availability_status matches what we're doing (available or unavailable)
      1. `POST /sources/:id/check_availability` -- request an availability check for that source
   2. Else: continue the loop

It does the availbility checking requests in parallel, using the `choke` unbuffered channel limiting the amount of in-flight requests at once. Currently (8/30/21) that number is set to 3 reqs at once.

### Dev Info

Main Application logic is in `main.go`, the response struct (and other types in the future) will live in `types.go`. A `Makefile` is provided for easy building/running.

In order to run without minikube/ephemeral environment - one just needs to provide the `SOURCES_*` variables like in the clowdapp template:
- `SOURCES_SCHEME`
- `SOURCES_HOST`
- `SOURCES_PORT`
- `SOURCES_PSK`

Once ran the output is similar to this:
```text
$ SOURCES_HOST=minikube.local SOURCES_PORT=80 SOURCES_PSK=thisMustBeEphOrMinikube make run
go build
./sources-monitor-go -status all
2021/08/30 10:35:16 Checking sources with [all] status from [http://minikube.local:80]
2021/08/30 10:35:16 Requesting [limit 100] + [offset 0] status from internal API at [http://minikube.local:80]
2021/08/30 10:35:16 Requesting availability status for [tenant 6089719], [source 3]
2021/08/30 10:35:16 Requesting availability status for [tenant 6089719], [source 2]
2021/08/30 10:35:16 Requesting availability status for [tenant 6089719], [source 1]
..... ( lots of lines )
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 103]
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 106]
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 107]
2021/08/30 10:35:23 Requesting [limit 100] + [offset 100] status from internal API at [http://minikube.local:80]
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 108]
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 109]
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 110]
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 111]
2021/08/30 10:35:23 Requested availability for 104 sources, waiting for all routines to complete...
2021/08/30 10:35:23 Requesting availability status for [tenant 6089719], [source 112]
```

## License

This project is available as open source under the terms of the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).
