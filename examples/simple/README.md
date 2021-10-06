# Instructions

In this example we deploy an application group consisting of two demo applications,

- Istio bookinfo app (with subcharts) : [source](https://istio.io/latest/docs/examples/bookinfo/)
- Ambassador : [source](https://www.getambassador.io/)

## Prerequisites

- `kubectl`

Install the `ApplicationGroup`:

```terminal
kubectl apply -f examples/simple/bookinfo.yaml

applicationgroup.orkestra.azure.microsoft.com/bookinfo created
```

The orkestra controller logs should look as follows on success,

```log
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T07:53:24.452Z       INFO    setup   starting manager
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T07:53:24.453Z       INFO    controller-runtime.manager      starting metrics server {"path": "/metrics"}
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T07:53:24.453Z       INFO    controller-runtime.controller   Starting EventSource    {"controller": "applicationgroup", "source": "kind source: /, Kind="}
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T07:53:24.554Z       INFO    controller-runtime.controller   Starting Controller     {"controller": "applicationgroup"}
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T07:53:24.554Z       INFO    controller-runtime.controller   Starting workers        {"controller": "applicationgroup", "worker count": 1}
... truncated for brevity ...
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T08:04:18.875Z       DEBUG   controllers.ApplicationGroup    workflow ran to completion and succeeded        {"appgroup": "bookinfo"}
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T08:04:18.901Z       DEBUG   controller-runtime.controller   Successfully Reconciled {"controller": "applicationgroup", "request": "/bookinfo"}
orkestra-885c5ff4-kh7n9 orkestra 2021-03-23T08:04:18.902Z       DEBUG   controller-runtime.manager.events       Normal  {"object": {"kind":"ApplicationGroup","name":"bookinfo","uid":"52c5095e-0aa1-4067-a434-f1155ebbbdcd","apiVersion":"orkestra.azure.microsoft.com/v1alpha1","resourceVersion":"30145"}, "reason": "ReconcileSuccess", "message": "Successfully reconciled ApplicationGroup bookinfo"}
```

(_optional_) The Argo dashboard should show the DAG nodes in Green 

![workflow](workflow.png)

### Verify that the Application helm release have been successfully deployed

```shell
helm ls

NAME            NAMESPACE       REVISION        UPDATED                                 STATUS    CHART            APP VERSION
orkestra        orkestra        1               2021-03-23 08:02:15.0044864 +0000 UTC   deployed  orkestra-0.1.0   0.1.0
ambassador      ambassador      1               2021-03-23 08:02:35.0044864 +0000 UTC   deployed  ambassador-6.6.0 1.12.1     
bookinfo        bookinfo        1               2021-03-23 08:04:08.6088786 +0000 UTC   deployed  bookinfo-v1      0.16.2     
details         bookinfo        1               2021-03-23 08:03:26.1043919 +0000 UTC   deployed  details-v1       1.16.2     
productpage     bookinfo        1               2021-03-23 08:03:47.4150589 +0000 UTC   deployed  productpage-v1   1.16.2     
ratings         bookinfo        1               2021-03-23 08:03:25.9770024 +0000 UTC   deployed  ratings-v1       1.16.2     
reviews         bookinfo        1               2021-03-23 08:03:36.9634599 +0000 UTC   deployed  reviews-v1       1.16.2     
```

## Send request to `productpage` via Ambassador gateway/proxy

```terminal
kubectl -n default exec curl -- curl -ksS https://ambassador.ambassador:443/bookinfo/ | grep -o "<title>.*</title>"
<title>Simple Bookstore App</title>
```
