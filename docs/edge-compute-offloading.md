Computation offloading from the Edge
===================

## Motivation

Edge clusters have limited compute capabilities. For this reason, it is desirable to have
a system which automatically offloads workloads from the Edge to the Cloud.

## Goals

* Automatically offload computational workload from the Edge to the Cloud.
* Provide a flexible way to replace edge offload functionality with different scalers.

## Non-Goals

* Don't interfere with the [gradual rollout](https://knative.dev/docs/serving/rolling-out-latest-revision/) feature in Knative serving.

## Proposal

The proposal is made up of two main parts:

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
    # TODO: check if this is necessary
    # this is added so the revision is not removed by the garbage collector
    serving.knative.dev/no-gc: true
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

In order to dynamically route the traffic towards the Cloud, a separate controller
will watch all the reverse proxy revisions and keep a list of each of their
services.

Periodically, the controller will scrape the metrics of each of the service in order
to judge the compute workload on the Edge. When the controller notices that the Edge
cannot support the workload, it will start routing more traffic towards the reverse
proxy revision.

Let's say we start with the following service:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: example-service
  namespace: default
  ...
spec:
  ...
  traffic:
    - latestRevision: true
      percent: 100
    - name: example-service-edge-compute-offload
      tag: edge-compute-offload-1
      percent: 0
```

The controller will increase the percentage of `example-service-edge-compute-offload`
until the metrics stabilize under the upper threshold. After the metrics stabilize, the
service might look like this:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: example-service
  namespace: default
  ...
spec:
  ...
  traffic:
    - latestRevision: true
      percent: 20
    - name: example-service-edge-compute-offload
      tag: edge-compute-offload-1
      percent: 80
```

After a cooldown period, the controller will gradually decrease the traffic percentage
to the reverse proxy. This is done until the metrics are stabilizing around the
lower threshold.

#### Scraped metrics

The following metrics are scraped by the controller:

1. TODO

#### Decision algorithm

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
