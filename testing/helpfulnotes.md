# Helpful Notes for writing tests with brigade
brig v1 api


# Getting Started

In my testing I have found Brigade works best with Kind. Before you begin make sure docker and kind are running. 

You will need to apply the binding.yml file to give the bridge-worker pods the ability to use kubectl commands. The Dockerfile will be the image the brigade job is run on. For now the brigade.js does the setup and grabs all the dependencies.

You must install Brigade onto your cluster. [Brigade Install Guide](https://docs.brigade.sh/intro/install/)
For this guide we are using the Brig CLI tool so please download that as well. 

```

kubectl get pods -A

```

You should now see the following brigade pods running

```
NAMESPACE            NAME                                             READY   STATUS      RESTARTS   AGE
default              brigade-server-brigade-api-7656489497-xczb7      1/1     Running     0          3m23s
default              brigade-server-brigade-ctrl-9d678c8bc-4h6nf      1/1     Running     0          3m23s
default              brigade-server-brigade-vacuum-1619128800-q24dh   0/1     Completed   0          34s
default              brigade-server-kashti-6ff4d6c99c-2dg87           1/1     Running     0          3m23s

```

Using brig we will create a sample project.

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

kubectl apply -f binding.yml

```
Now we can run our brigade.js file on our cluster to verify orkestra is working.

```
cd testing
brig run -f brigade.js mysampleproject
```

To check logs of your Jobs use,

```
brig build logs --last --jobs

```

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

