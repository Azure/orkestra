# Roadmap

## Future Integrations

Popular frameworks offered as plugins and/or core components of the Orkestra controlplane

### Progressive Delivery (Canary/Blue-Green rollouts)

Using popular open source "Progress Delivery Frameworks" as plugins, perform progressive delivery of Applications in the Application Group.

Utilize service-meshes (e.g. [Istio](https://istio.io/)) under the hood to provide "Automated Canary Analysis" during Canary upgrades by using "Traffic Shaping" features provided by the Service Mesh.

#### [Argo Rollouts](https://argoproj.github.io/argo-rollouts/)

Generate [`Rollout`](https://argoproj.github.io/argo-rollouts/features/specification/) resources on the fly by parsing the `Deployment` and `Service` resources in the application (or subcharts) templates.

Ability to *inline* the [`AnalysisTemplate`](https://argoproj.github.io/argo-rollouts/features/analysis.html#analysis-progressive-delivery) config and strategy configurations as part of the `ApplicationSpec`.

#### [FluxCD Flagger](https://flagger.app/)

Generate [`Canary`](https://docs.flagger.app/usage/how-it-works#canary-resource) resources on the fly by parsing the `Deployment` and `Service` resources in the application (or subcharts) templates.

Ability to *inline* the Analysis Template specs and strategy configurations as part of the `ApplicationSpec`.

### [Keptn](https://keptn.sh/)

Keptn as a plugin (or component *TBD*) of Orkestra control-plane.
Keptn brings:

- Observability, dashboards & alerting
- SLO-driven multi-stage delivery
- Operations & remediation
- **Chaos Testing**
, as part of the application rollout process providing finer-grained control over promotions as you progress through the DAG workflow.

## Examples

### CNCF's delivery application '[Podtato-head](https://github.com/cncf/podtato-head)' with an ingress controller (*TBD*)

*TBD*

### Argo Rollouts

Example using podtato-head application with [Argo Rollouts](https://github.com/cncf/podtato-head/tree/main/delivery/rollout) templates.

### Custom Executor

Runs custom queries against prometheus without the use of Service Mesh (or progressive delivery frameworks) or Automated Canary Analysis, powered through the BYO executor container approach.

### Keptn

*TBD*
