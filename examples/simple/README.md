# Instructions

In this example we deploy an application group consisting of two demo applications,
- kafka (and it's dependent chart - zookeeper)
- redis

## Prerequisites

- `kubectl`

Install the `ApplicationGroup` and associated `Application` CRDs:

```console
kubectl apply -f examples/simple

applicationgroup.orkestra.azure.microsoft.com/dev created
application.orkestra.azure.microsoft.com/kafka-dev created
application.orkestra.azure.microsoft.com/redis-dev created
```

The orkestra controller logs should look as follows on success,

```log
2021-02-03T09:21:18.395Z        DEBUG   controller-runtime.manager.events       Normal  {"object": {"kind":"Application","name":"redis-dev","uid":"6ffc6c9d-1343-4911-8683-a60b5bbdf28d","apiVersion":"orkestra.azure.microsoft.com/v1alpha1","resourceVersion":"61624"}, "reason": "ReconcileSuccess", "message": "Successfully reconciled Application redis-dev"}
2021-02-03T09:21:18.395Z        DEBUG   controller-runtime.controller   Successfully Reconciled {"controller": "application", "request": "/kafka-dev"}
2021-02-03T09:21:18.395Z        DEBUG   controller-runtime.controller   Successfully Reconciled {"controller": "application", "request": "/redis-dev"}
2021-02-03T09:21:20.314Z        DEBUG   controller-runtime.controller   Successfully Reconciled {"controller": "applicationgroup", "request": "/dev"}
2021-02-03T09:21:20.314Z        DEBUG   controller-runtime.manager.events       Normal  {"object": {"kind":"ApplicationGroup","name":"dev","uid":"6d9ba709-70f0-438e-ad40-f4be7376b0f5","apiVersion":"orkestra.azure.microsoft.com/v1alpha1","resourceVersion":"61634"}, "reason": "ReconcileSuccess", "message": "Successfully reconciled ApplicationGroup dev"}
2021-02-03T09:21:20.325Z        DEBUG   controller-runtime.controller   Successfully Reconciled {"controller": "applicationgroup", "request": "/dev"}
2021-02-03T09:21:20.325Z        DEBUG   controller-runtime.manager.events       Normal  {"object": {"kind":"ApplicationGroup","name":"dev","uid":"6d9ba709-70f0-438e-ad40-f4be7376b0f5","apiVersion":"orkestra.azure
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