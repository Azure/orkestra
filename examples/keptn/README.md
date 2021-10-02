# Manual Testing

## Authenticate with keptn

```terminal
export KEPTN_API_TOKEN=$(kubectl get secret keptn-api-token -n orkestra -ojsonpath='{.data.keptn-api-token}' | base64 --decode)
export KEPTN_ENDPOINT=http://$(kubectl get svc api-gateway-nginx -n orkestra -ojsonpath='{.status.loadBalancer.ingress[0].ip}')/api
```

```terminal
keptn auth --endpoint=$KEPTN_ENDPOINT --api-token=$KEPTN_API_TOKEN

Starting to authenticate
Successfully authenticated against the Keptn cluster http://20.72.120.233/api
```

## Retrieve username and password for Keptn bridge (dashboard)

```terminal
keptn configure bridge --output   
```

## Trigger evaluation

```terminal
keptn create project hey --shipyard=./shipyard.yaml
keptn create service bookinfo --project=hey
keptn configure monitoring prometheus --project=hey --service=bookinfo
keptn add-resource --project=hey --service=bookinfo --resource=slo.yaml --resourceUri=slo.yaml --stage=dev
keptn add-resource --project=hey --service=bookinfo --resource=prometheus/sli.yaml  --resourceUri=prometheus/sli.yaml --stage=dev
keptn add-resource --project=hey --service=bookinfo --resource=job/config.yaml  --resourceUri=job/config.yaml --stage=dev
keptn trigger evaluation --project=hey --service=bookinfo --timeframe=5m --stage dev --start $(date -u +"%Y-%m-%dT%T")
```
