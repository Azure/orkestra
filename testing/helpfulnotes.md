# Helpful Notes for writing tests with brigade
brig v1 api

In my testing I have found Brigade works best with Kind. Before you begin make sure docker and kind are running. 

You will need to apply the binding.yml file to give the bridge-worker pods the ability to use kubectl commands. The Dockerfile will be the image the brigade job is run on. For now the brigade.js does the setup and grabs all the dependencies.


To check logs of your Jobs use,

```
brig build logs --last --jobs

```

To install the brigadecore-utils at runtime add the --config flag to brig run with the brigade.json file

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

