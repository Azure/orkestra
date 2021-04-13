# Helpful Notes for writing tests with brigade
brig v1 api

To check logs of your Jobs use,

```
brig build logs --last --jobs

```

To install the brigadecore-utils at runtime add the --config flag to brig run with the brigade.json file

```

brig run <Project> --file brigade.js --config brigade.json
 
```
