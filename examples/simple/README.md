# Instructions

In this example we deploy an application group consisting of two demo applications,
- kafka (and it's dependent chart - zookeeper)
- redis

## Prerequisites

- `kubectl`

Install the `ApplicationGroup`: 

```console
kubectl apply -f examples/simple

applicationgroup.orkestra.azure.microsoft.com/dev created
```

The orkestra controller logs should look as follows on success,

```log
2021-03-02T19:06:53.729Z        INFO    controller-runtime.metrics      metrics server is starting to listen    {"addr": ":8080"}
2021-03-02T19:06:56.942Z        INFO    setup   starting manager
2021-03-02T19:06:56.944Z        INFO    controller-runtime.controller   Starting EventSource    {"controller": "applicationgroup", "source": "kind source: /, Kind="}
2021-03-02T19:06:56.945Z        INFO    controller-runtime.controller   Starting Controller     {"controller": "applicationgroup"}
2021-03-02T19:06:56.946Z        INFO    controller-runtime.manager      starting metrics server {"path": "/metrics"}
2021-03-02T19:06:57.046Z        INFO    controller-runtime.controller   Starting workers        {"controller": "applicationgroup", "worker count": 1}
2021-03-02T19:06:59.957Z        DEBUG   controller-runtime.manager.events       Warning {"object": {"kind":"ApplicationGroup","name":"dev","uid":"7e0382cc-843c-4a55-8e5d-b6984db10ed5","apiVersion":"orkestra.azure.microsoft.com/v1alpha1","resourceVersion":"879"}, "reason": "ReconcileError", "message": "Failed to reconcile ApplicationGroup dev with Error failed to DELETE argo workflow object : resource name may not be empty"}
2021-03-02T19:07:00.957Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:07:05.968Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:07:10.984Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:07:15.997Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:07:21.009Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:07:25.161Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:07:26.025Z        DEBUG   controllers.ApplicationGroup    workflow in pending/running state. requeue and reconcile after a short period        {"appgroup": "dev", "phase": "Running", "status-error": ""}
2021-03-02T19:08:01.195Z        DEBUG   controller-runtime.manager.events       Normal  {"object": {"kind":"ApplicationGroup","name":"dev","uid":"7e0382cc-843c-4a55-8e5d-b6984db10ed5","apiVersion":"orkestra.azure.microsoft.com/v1alpha1","resourceVersion":"1345"}, "reason": "ReconcileSuccess", "message": "Successfully reconciled ApplicationGroup dev"}
```

(_optional_) The Argo dashboard should show the DAG nodes in Green 

<p align="center"><img src="./workflow.png" width="750x" /></p>

**Verify that the Application helm release have been successfully deployed**

```console
helm ls

NAME                    NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
orkestra                orkestra        1               2021-02-03 00:53:59.4021554 -0800 PST   deployed        orkestra-0.1.0  0.1.0
orkestra-kafka-dev      orkestra        1               2021-02-03 09:22:10.6098917 +0000 UTC   deployed        kafka-12.4.1    2.7.0
orkestra-redis-dev      orkestra        1               2021-02-03 09:21:29.2499689 +0000 UTC   deployed        redis-12.2.3    6.0.9
orkestra-zookeeper      orkestra        1               2021-02-03 09:21:46.9612055 +0000 UTC   deployed        zookeeper-6.2.1 3.6.2
```