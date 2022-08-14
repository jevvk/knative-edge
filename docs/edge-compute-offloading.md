Computation offloading from the Edge
===================

## Motivation

Edge clusters have limited compute capabilities. For this reason, it is desirable to have
a system which automatically offloads workloads from the Edge to the Cloud.

## Goals

* Automatically offload computational workload from the Edge to the Cloud.
* Provide a flexible way to replace edge offload functionality with different scalers.

# Non-Goals

* Don't interfere with the [gradual rollout](https://knative.dev/docs/serving/rolling-out-latest-revision/) feature in Knative serving.

## Proposal

:

1. setting up and managing the cloud-hosted service reverse proxy
2. controlling the revision traffic percentage dynamically 

### Reverse proxy

The reverse proxy component is a simple proxy written in go. It has the following
requirements:

1. it should respect `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables
2. it should forward requests to the cloud-hosted service using `REMOTE_URL` environment 
variables

### Revision management

In order to enable the reverse proxy, a revision is created for each compute offloaded 
service. This is done by watching Knative services for the annotation 
`edge.jevv.dev/offload-to-remote`. Let's say we start with the following service:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: example-service
  namespace: default
  generation: 1
  annotations:
    ...
    edge.jevv.dev/offload-to-remote: "true"
spec:
  ...
```

When the service is mirrored from the edge to the cloud, a controller will create a 
revision that looks like this:

```yaml
apiVersion: serving.knative.dev/v1
kind: Revision
metadata:
  name: example-service-edge-compute-offload
  namespace: default
  generation: 1
  annotations:
    serving.knative.dev/no-gc: true # done for the revision
  ownerReferences: 
    # TODO: check if setting the service as the owner is ok
    #       knative-serving uses the configuration (which is managed by the service)
    - apiVersion: serving.knative.dev/v1
      kind: Service
      controller: true
      name: example-service
      uuid: xxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
spec:
  containerConcurrency: 0
  enableServiceLinks: false
  timeoutSeconds: 30
  containers:
    - name: proxy
      image: ko://edge.jevv.dev/proxy
      env:
        - name: REMOTE_URL
          # this example uses the internal domain, actual value will depend on
          # the domain mapping that was set
          value: http://example-service.default.svc.cluster.local
        - name: REMOTE_TIMEOUT
          value: 30s
        
```

After the revision is created, the original service is updated in order for the Knative 
route to be created.

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: example-service
  namespace: default
  generation: 2
  labels:
    ...
    # TODO: check which labels should be created for knative-serving
    edge.jevv.dev/managed: "true"
    edge.jevv.dev/edge-local: "true"
    app.kubernetes.io/managed-by: "knative-edge"
    app.kubernetes.io/created-by: "knative-edge-controller"
  annotations:
    ...
    # TODO: check which annotations should be created for knative-serving
    edge.jevv.dev/offload-to-remote: "true"
    edge.jevv.dev/last-generation: "1"
spec:
  ...
  traffic:
    - latestRevision: true
      percent: 100
    - name: example-service-edge-compute-offload
      tag: edge-compute-offload-1
      percent: 0
```



### Dynamic offloading

TODO

## Example

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: example-service
  namespace: default
  annotations:
    ...
    edge.jevv.dev/offload-to-remote: "true"
spec:
...
```



Rough idea:
- create special revision for offloading to edge
- the revision will listen to http requests and pass them to the remote
- knative edge listens to the knative services metrics and changes the traffic ratio accordingly
  - need to define which metrics to listen to
  - either use the knative metrics or node metrics

Potential issues:
- gradual rollouts
- edge offload revision being changed by knative controller(s)

Example:
```
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: example-service
  namespace: default
  annotations:
    edge.jevv.dev/offload-to-remote: true
spec:
  template:
    spec:
      ...
      traffic:
      - latestRevision: true
        percent: 100
      - name: knative-edge-offload
        percent: 0 # changed depending on the load
---
apiVersion: serving.knative.dev/v1
kind: Revision
metadata:
  name: knative-edge-offload
  namespace: default
  annotations:
    serving.knative.dev/no-gc: "true"
spec:
...
```