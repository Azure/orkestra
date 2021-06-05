---
layout: default
title: Operators 
nav_order: 4
---
# Helm Values

```yaml
# service account to be used by all orkestra components
serviceAccount: orkestra
# number of pod replicas to run (single leader is elected using leader-election)
replicaCount: 1
# path of the directory used to store the pulled helm charts
chartStorePath: "/etc/orkestra/charts/pull"

image:
  # image docker repository/registry
  repository: azureorkestra/orkestra
  # image pull policy
  pullPolicy: Always 
  # image docker tag
  tag: "latest"

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: *serviceAccount

podAnnotations: {}

podSecurityContext: {}

securityContext: {}

resources: {}

nodeSelector: {}

tolerations: []

affinity: {}

# Remediation settings on failure during installation or upgrades
remediation:
  # If set to true prevents the controller from deleting the failed resources
  # For development use only!
  disabled: false

# Cleanup the pulled helm charts from the local storage directory
cleanup:
  enabled: false

# logging debug level
debug:
  level: 5

chartmuseum:
  extraArgs:
    # - --storage-timestamp-tolerance 1s
  replicaCount: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
  image:
    repository: chartmuseum/chartmuseum
    tag: v0.12.0
    pullPolicy: IfNotPresent
  secret:
    labels: {}
  env:
    open:
      # storage backend, can be one of: local, alibaba, amazon, google, microsoft, oracle
      STORAGE: local
      # oss bucket to store charts for alibaba storage backend
      STORAGE_ALIBABA_BUCKET:
      # prefix to store charts for alibaba storage backend
      STORAGE_ALIBABA_PREFIX:
      # oss endpoint to store charts for alibaba storage backend
      STORAGE_ALIBABA_ENDPOINT:
      # server side encryption algorithm for alibaba storage backend, can be one
      # of: AES256 or KMS
      STORAGE_ALIBABA_SSE:
      # s3 bucket to store charts for amazon storage backend
      STORAGE_AMAZON_BUCKET:
      # prefix to store charts for amazon storage backend
      STORAGE_AMAZON_PREFIX:
      # region of s3 bucket to store charts
      STORAGE_AMAZON_REGION:
      # alternative s3 endpoint
      STORAGE_AMAZON_ENDPOINT:
      # server side encryption algorithm
      STORAGE_AMAZON_SSE:
      # gcs bucket to store charts for google storage backend
      STORAGE_GOOGLE_BUCKET:
      # prefix to store charts for google storage backend
      STORAGE_GOOGLE_PREFIX:
      # container to store charts for microsoft storage backend
      STORAGE_MICROSOFT_CONTAINER:
      # prefix to store charts for microsoft storage backend
      STORAGE_MICROSOFT_PREFIX:
      # container to store charts for openstack storage backend
      STORAGE_OPENSTACK_CONTAINER:
      # prefix to store charts for openstack storage backend
      STORAGE_OPENSTACK_PREFIX:
      # region of openstack container
      STORAGE_OPENSTACK_REGION:
      # path to a CA cert bundle for your openstack endpoint
      STORAGE_OPENSTACK_CACERT:
      # compartment id for for oracle storage backend
      STORAGE_ORACLE_COMPARTMENTID:
      # oci bucket to store charts for oracle storage backend
      STORAGE_ORACLE_BUCKET:
      # prefix to store charts for oracle storage backend
      STORAGE_ORACLE_PREFIX:
      # form field which will be queried for the chart file content
      CHART_POST_FORM_FIELD_NAME: chart
      # form field which will be queried for the provenance file content
      PROV_POST_FORM_FIELD_NAME: prov
      # levels of nested repos for multitenancy. The default depth is 0 (singletenant server)
      DEPTH: 0
      # show debug messages
      DEBUG: false
      # output structured logs as json
      LOG_JSON: true
      # disable use of index-cache.yaml
      DISABLE_STATEFILES: false
      # disable Prometheus metrics
      DISABLE_METRICS: true
      # disable all routes prefixed with /api
      DISABLE_API: true
      # allow chart versions to be re-uploaded
      ALLOW_OVERWRITE: false
      # absolute url for .tgzs in index.yaml
      CHART_URL:
      # allow anonymous GET operations when auth is used
      AUTH_ANONYMOUS_GET: false
      # sets the base context path
      CONTEXT_PATH:
      # parallel scan limit for the repo indexer
      INDEX_LIMIT: 0
      # cache store, can be one of: redis (leave blank for inmemory cache)
      CACHE:
      # address of Redis service (host:port)
      CACHE_REDIS_ADDR:
      # Redis database to be selected after connect
      CACHE_REDIS_DB: 0
      # enable bearer auth
      BEARER_AUTH: false
      # auth realm used for bearer auth
      AUTH_REALM:
      # auth service used for bearer auth
      AUTH_SERVICE:
    field:
      # POD_IP: status.podIP
    secret:
      # username for basic http authentication
      BASIC_AUTH_USER:
      # password for basic http authentication
      BASIC_AUTH_PASS:
      # GCP service account json file
      GOOGLE_CREDENTIALS_JSON:
      # Redis requirepass server configuration
      CACHE_REDIS_PASSWORD:
    # Name of an existing secret to get the secret values ftom
    existingSecret:
    # Stores Enviromnt Variable to secret key name mappings
    existingSecretMappings:
      # username for basic http authentication
      BASIC_AUTH_USER:
      # password for basic http authentication
      BASIC_AUTH_PASS:
      # GCP service account json file
      GOOGLE_CREDENTIALS_JSON:
      # Redis requirepass server configuration
      CACHE_REDIS_PASSWORD:

  deployment:
    # Define scheduler name. Use of 'default' if empty
    schedulerName: ""
    ## Chartmuseum Deployment annotations
    annotations: {}
    #   name: value
    labels: {}
    #   name: value
    matchlabels: {}
    #   name: value
  replica:
    ## Chartmuseum Replicas annotations
    annotations: {}
    ## Read more about kube2iam to provide access to s3 https://github.com/jtblin/kube2iam
    #   iam.amazonaws.com/role: role-arn
  service:
    servicename:
    type: ClusterIP
    externalTrafficPolicy: Local
    ## Limits which cidr blocks can connect to service's load balancer
    ## Only valid if service.type: LoadBalancer
    loadBalancerSourceRanges: []
    # clusterIP: None
    externalPort: 8080
    nodePort:
    annotations: {}
    labels: {}

  serviceMonitor:
    enabled: false
    # namespace: prometheus
    labels: {}
    metricsPath: "/metrics"
    # timeout: 60
    # interval: 60

  resources: {}
  #  limits:
  #    cpu: 100m
  #    memory: 128Mi
  #  requests:
  #    cpu: 80m
  #    memory: 64Mi

  probes:
    liveness:
      initialDelaySeconds: 5
      periodSeconds: 10
      timeoutSeconds: 1
      successThreshold: 1
      failureThreshold: 3
    readiness:
      initialDelaySeconds: 5
      periodSeconds: 10
      timeoutSeconds: 1
      successThreshold: 1
      failureThreshold: 3

  serviceAccount:
    create: false
    # name:
    ## Annotations for the Service Account
    annotations: {}

  # UID/GID 1000 is the default user "chartmuseum" used in
  # the container image starting in v0.8.0 and above. This
  # is required for local persistent storage. If your cluster
  # does not allow this, try setting securityContext: {}
  securityContext:
    enabled: true
    fsGroup: 1000
    ## Optionally, specify supplementalGroups and/or
    ## runAsNonRoot for security purposes
    # runAsNonRoot: true
    # supplementalGroups: [1000]

  containerSecurityContext: {}

  priorityClassName: ""

  nodeSelector: {}

  tolerations: []

  affinity: {}

  persistence:
    enabled: false
    accessMode: ReadWriteOnce
    size: 8Gi
    labels: {}
    path: /storage
    #   name: value
    ## A manually managed Persistent Volume and Claim
    ## Requires persistence.enabled: true
    ## If defined, PVC must be created manually before volume will be bound
    # existingClaim:

    ## Chartmuseum data Persistent Volume Storage Class
    ## If defined, storageClassName: <storageClass>
    ## If set to "-", storageClassName: "", which disables dynamic provisioning
    ## If undefined (the default) or set to null, no storageClassName spec is
    ##   set, choosing the default provisioner.  (gp2 on AWS, standard on
    ##   GKE, AWS & OpenStack)
    ##
    # storageClass: "-"
    # volumeName:
    pv:
      enabled: false
      pvname:
      capacity:
        storage: 8Gi
      accessMode: ReadWriteOnce
      nfs:
        server:
        path:

  ## Init containers parameters:
  ## volumePermissions: Change the owner of the persistent volume mountpoint to RunAsUser:fsGroup
  ##
  volumePermissions:
    image:
      registry: docker.io
      repository: bitnami/minideb
      tag: buster
      pullPolicy: Always
      ## Optionally specify an array of imagePullSecrets.
      ## Secrets must be manually created in the namespace.
      ## ref: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
      ##
      # pullSecrets:
      #   - myRegistryKeySecretName

  ## Ingress for load balancer
  ingress:
    enabled: false
  ## Chartmuseum Ingress labels
  ##
  #   labels:
  #     dns: "route53"

  ## Chartmuseum Ingress annotations
  ##
  #   annotations:
  #     kubernetes.io/ingress.class: nginx
  #     kubernetes.io/tls-acme: "true"

  ## Chartmuseum Ingress hostnames
  ## Must be provided if Ingress is enabled
  ##
  #  hosts:
  #    - name: chartmuseum.domain1.com
  #      path: /
  #      tls: false
  #    - name: chartmuseum.domain2.com
  #      path: /
  #
  #      ## Set this to true in order to enable TLS on the ingress record
  #      tls: true
  #
  #      ## If TLS is set to true, you must declare what secret will store the key/certificate for TLS
  #      ## Secrets must be added manually to the namespace
  #      tlsSecret: chartmuseum.domain2-tls

  # Adding secrets to tiller is not a great option, so If you want to use an existing
  # secret that contains the json file, you can use the following entries
  gcp:
    secret:
      enabled: false
      # Name of the secret that contains the encoded json
      name:
      # Secret key that holds the json value.
      key: credentials.json
  oracle:
    secret:
      enabled: false
      # Name of the secret that contains the encoded config and key
      name:
      # Secret key that holds the oci config
      config: config
      # Secret key that holds the oci private key
      key_file: key_file
  bearerAuth:
    secret:
      enabled: false
      publicKeySecret: chartmuseum-public-key

helm-controller:
  # Default values for helm-controller.
  # This is a YAML-formatted file.
  # Declare variables to be passed into your templates.
  concurrent: 10

  replicaCount: 1

  image:
    repository: fluxcd/helm-controller
    pullPolicy: Always 
    # Overrides the image tag whose default is the chart appVersion.
    tag: "v0.9.0"

  imagePullSecrets: []
  nameOverride: ""
  fullnameOverride: ""
  containerName: manager

  serviceAccount:
    # Specifies whether a service account should be created
    create: true
    # Annotations to add to the service account
    annotations: {}
    # The name of the service account to use.
    # If not set and create is true, a name is generated using the fullname template
    name: "orkestra"
    
  podAnnotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"

  podSecurityContext: {}
    # fsGroup: 2000

  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true

  resources: {}
    # We usually recommend not to specify default resources and to leave this as a conscious
    # choice for the user. This also increases chances charts run on environments with little
    # resources, such as Minikube. If you do want to specify resources, uncomment the following
    # lines, adjust them as necessary, and remove the curly braces after 'resources:'.
    # limits:
    #   cpu: 100m
    #   memory: 128Mi
    # requests:
    #   cpu: 100m
    #   memory: 128Mi

  source-controller:
    serviceAccount:
      create: false
      name: "orkestra"
```
