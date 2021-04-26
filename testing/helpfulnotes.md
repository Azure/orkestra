# Testing Environment Setup

## Getting Started

### Prerequisites
* `Argo` - Argo workflow client (Follow the instructions to install the binary from [releases](https://github.com/argoproj/argo/releases))
* `Brigade` - [brigade install guide](https://docs.brigade.sh/intro/install/)
* `brig` - [brig guide](https://docs.brigade.sh/topics/brig/)
* `kubectl` - [kubectl install guide](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
* A Kubernetes Cluster

When testing I used a KIND cluster but Brigade should work for minikube as well. Brigade docs have a section about Minikube and AKS, [found here](https://docs.brigade.sh/intro/install/#notes-for-minikube). 

Before you begin make sure docker and your cluster are running. 

The Dockerfile will be the image the brigade job is run on, at the moment this Docker image is not used but should be uploaded to DockerHub so our brigade.js can download it as an image. For now the brigade.js does the setup and grabs all the dependencies. After installing Brigade, you should now see the following brigade pods running

```
helm install brigade-server brigade/brigade
kubectl get pods -A
NAMESPACE            NAME                                             READY   STATUS      RESTARTS   AGE
default              brigade-server-brigade-api-7656489497-xczb7      1/1     Running     0          3m23s
default              brigade-server-brigade-ctrl-9d678c8bc-4h6nf      1/1     Running     0          3m23s
default              brigade-server-brigade-vacuum-1619128800-q24dh   0/1     Completed   0          34s
default              brigade-server-kashti-6ff4d6c99c-2dg87           1/1     Running     0          3m23s
```

Using brig we will create a sample project. For our testing we just use all the defaults. The brigade.js path for us would be `testing/brigade.js`.

```
brig project create
? VCS or no-VCS project? no-VCS
? Project Name mysampleproject
? Add secrets? No
? Secret for the Generic Gateway (alphanumeric characters only). Press Enter if you want it to be auto-generated [? for ? Secret for the Generic Gateway (alphanumeric characters only). Press Enter if you want it to be auto-generated
Auto-generated Generic Gateway Secret: FPK8O
? Default script ConfigMap name
? Upload a default brigade.js script <PATH_TO_BRIGADE.js>
? Default brigade.json config ConfigMap name
? Upload a default brigade.json config
? Configure advanced options No
```

Confirm your sample project was created,

```
brig project list 
NAME            ID                                                              REPO
mysampleproject brigade-a50ed8c1dbd7fa803b75f009f893b56bfd12347cadb1e404c12  github.com/brigadecore/empty-testbed
```

To give our brigade jobs the ability to access our kubectl commands we have to apply the binding.yml file onto our cluster. This file gives the brigade jobs permissions for various kubectl commands.

```
cd testing
kubectl apply -f binding.yml
```

We also want to run the argo server so we can view the workflow, and so our validation tests can check if the workflow pods were deployed successfully. 

```
argo server
```

Now we can run our brigade.js file on our cluster to verify orkestra is working.

```
cd testing
brig run -f brigade.js mysampleproject
Event created. Waiting for worker pod named "brigade-worker-01f47mb971tp4f3k6erx8fxhrr".
Build: 01f47mb971tp4f3k6erx8fxhrr, Worker: brigade-worker-01f47mb971tp4f3k6erx8fxhrr
prestart: no dependencies file found
[brigade] brigade-worker version: 1.2.1
[brigade:k8s] Creating PVC named brigade-worker-01f47mb971tp4f3k6erx8fxhrr
Running on exec
[brigade:k8s] Creating secret test-runner-01f47mb971tp4f3k6erx8fxhrr
[brigade:k8s] Creating pod test-runner-01f47mb971tp4f3k6erx8fxhrr
[brigade:k8s] Timeout set at 1500000 milliseconds
[brigade:k8s] Pod not yet scheduled
[brigade:k8s] default/test-runner-01f47mb971tp4f3k6erx8fxhrr phase Pending
[brigade:k8s] default/test-runner-01f47mb971tp4f3k6erx8fxhrr phase Running
done
[brigade:k8s] default/test-runner-01f47mb971tp4f3k6erx8fxhrr phase Running
```

Upon completion of the test runner we should see,
```
[brigade:k8s] default/test-runner-01f47mb971tp4f3k6erx8fxhrr phase Running
done
[brigade:k8s] default/test-runner-01f47mb971tp4f3k6erx8fxhrr phase Succeeded
done
[brigade:app] after: default event handler fired
[brigade:app] beforeExit(2): destroying storage
[brigade:k8s] Destroying PVC named brigade-worker-01f47mb971tp4f3k6erx8fxhrr
```

To check the logs of the test runner and validations,

```
brig build logs --last --jobs
```

Any errors will be output to a default log file, `log.txt` in the testing folder.

If you need to install the brigadecore-utils at runtime add the --config flag to brig run with the brigade.json file

```
brig run <PROJECT_NAME> --file brigade.js --config brigade.json
```

(Unnecessary since we are not using KindJob anymore) The KindJob object in the Brigade API requires you to allow mount hosts in the project. When creating your project with 

```
brig project create
```

Enter Y when asked for advanced options, this will allow you to set allow mount hosts to true.


## Known Issues

There is a docker related bug tracked here: [issue 5593](https://github.com/docker/for-win/issues/5593), which causes there to be time drift when using Docker for Windows. This prevents debian images from properly installing packages since the system clock is wrong. 

Quick fix: Restart computer or restart docker

