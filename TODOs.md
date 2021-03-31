# Things to do to support `HelmRelease`s from helm-controller

## Functional changes
- [ ] Replace generation of helm-operator's `HelmRelease` with helm-controller's `HelmRelease` + `HelmRepository` resources
- [ ] (*) Deprecate registry-config being configured by Orkestra helm chart's `values.yaml`. We no longer want this to be static
as it should be dynamically configurable per `ApplicationGroup` resources
- [ ] Leverage helm-controller's **remediation** and **rollback** features for remediation on failures/errors.

## API changes
- [ ] Modify `ApplicationGroup` spec to be configured with helm release configurations through native (new) fields instead of
embedding the `HelmRelease` object in the spec
- [ ] Modify `ApplicationGroup` spec to accept the helm registry configuration. See (*).
- [ ] Deprecate `GroupID` field

## Testing
- [ ] Add unit tests for crucial functions/methods like generation of Argo Workflows and `HelmRelease` & `HelmRepository` in the
workflow package (pkg/workflow).
- [ ] Integration tests using `EnvTest` using Behavior Driven Testing framework - Ginkgo (NOTE: We will not use the local etcd/apiserver
 but instead utilize a KIND cluster for testing - spun up on test init and torn down on test completion)
 