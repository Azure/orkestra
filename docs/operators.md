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

argo-workflows:
  images:
    # imagePullPolicy to apply to all containers
    pullPolicy: Always
    # Secrets with credentials to pull images from a private registry
    pullSecrets: []
    # - name: argo-pull-secret

  init:
    # By default the installation will not set an explicit one, which will mean it uses `default` for the namespace the chart is
    # being deployed to.  In RBAC clusters, that will almost certainly fail.  See the NOTES: section of the readme for more info.
    serviceAccount: ""

  createAggregateRoles: true

  ## String to partially override "argo-workflows.fullname" template
  ##
  nameOverride:

  ## String to fully override "argo-workflows.fullname" template
  ##
  fullnameOverride:

  ## Override the Kubernetes version, which is used to evaluate certain manifests
  ##
  kubeVersionOverride: ""

  # Restrict Argo to only deploy into a single namespace by apply Roles and RoleBindings instead of the Cluster equivalents,
  # and start argo-cli with the --namespaced flag. Use it in clusters with strict access policy.
  singleNamespace: false

  workflow:
    namespace: "" # Specify namespace if workflows run in another namespace than argo. This controls where the service account and RBAC resources will be created.
    serviceAccount:
      create: false # Specifies whether a service account should be created
      annotations: {}
      name: "argo-workflow" # Service account which is used to run workflows
    rbac:
      create: false # adds Role and RoleBinding for the above specified service account to be able to run workflows

  controller:
    image:
      registry: quay.io
      repository: argoproj/workflow-controller
      # Overrides the image tag whose default is the chart appVersion.
      tag: ""
    # parallelism dictates how many workflows can be running at the same time
    parallelism:
    # podAnnotations is an optional map of annotations to be applied to the controller Pods
    podAnnotations: {}
    # Optional labels to add to the controller pods
    podLabels: {}
    # SecurityContext to set on the controller pods
    podSecurityContext: {}
    # podPortName: http
    metricsConfig:
      enabled: false
      path: /metrics
      port: 9090
      servicePort: 8080
      servicePortName: metrics
    # the controller container's securityContext
    securityContext:
      readOnlyRootFilesystem: true
      runAsNonRoot: true
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
    persistence: {}
    # connectionPool:
    #   maxIdleConns: 100
    #   maxOpenConns: 0
    # # save the entire workflow into etcd and DB
    # nodeStatusOffLoad: false
    # # enable archiving of old workflows
    # archive: false
    # postgresql:
    #   host: localhost
    #   port: 5432
    #   database: postgres
    #   tableName: argo_workflows
    #   # the database secrets must be in the same namespace of the controller
    #   userNameSecret:
    #     name: argo-postgres-config
    #     key: username
    #   passwordSecret:
    #     name: argo-postgres-config
    #     key: password
    workflowDefaults: {} # Only valid for 2.7+
    #  spec:
    #    ttlStrategy:
    #      secondsAfterCompletion: 84600
    # workflowWorkers: 32
    # podWorkers: 32
    workflowRestrictions: {} # Only valid for 2.9+
    #  templateReferencing: Strict|Secure
    telemetryConfig:
      enabled: false
      path: /telemetry
      port: 8081
      servicePort: 8081
      servicePortName: telemetry
    serviceMonitor:
      enabled: false
      additionalLabels: {}
    serviceAccount:
      create: true
      name: ""
      # Annotations applied to created service account
      annotations: {}
    name: workflow-controller
    workflowNamespaces:
      - default
    containerRuntimeExecutor: docker
    instanceID:
      # `instanceID.enabled` configures the controller to filter workflow submissions
      # to only those which have a matching instanceID attribute.
      enabled: false
      # NOTE: If `instanceID.enabled` is set to `true` then either `instanceID.userReleaseName`
      # or `instanceID.explicitID` must be defined.
      # useReleaseName: true
      # explicitID: unique-argo-controller-identifier
    logging:
      level: info
      globallevel: "0"
    serviceType: ClusterIP
    # Annotations to be applied to the controller Service
    serviceAnnotations: {}
    # Optional labels to add to the controller Service
    serviceLabels: {}
    # Source ranges to allow access to service from. Only applies to
    # service type `LoadBalancer`
    loadBalancerSourceRanges: []
    resources: {}

    ## Extra environment variables to provide to the controller container
    ## extraEnv:
    ##   - name: FOO
    ##     value: "bar"
    extraEnv: []

    # Extra arguments to be added to the controller
    extraArgs: []
    replicas: 1
    pdb:
      enabled: false
      # minAvailable: 1
      # maxUnavailable: 1
    ## Node selectors and tolerations for server scheduling to nodes with taints
    ## Ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
    ##
    nodeSelector:
      kubernetes.io/os: linux
    tolerations: []
    affinity: {}
    # Leverage a PriorityClass to ensure your pods survive resource shortages
    # ref: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
    # PriorityClass: system-cluster-critical
    priorityClassName: ""
    # https://argoproj.github.io/argo-workflows/links/
    links: []
    clusterWorkflowTemplates:
      # Create a ClusterRole and CRB for the controller to access ClusterWorkflowTemplates.
      enabled: true

  # executor controls how the init and wait container should be customized
  executor:
    image:
      registry: quay.io
      repository: argoproj/argoexec
      # Overrides the image tag whose default is the chart appVersion.
      tag: ""
    resources: {}
    # Adds environment variables for the executor.
    env: {}
    # sets security context for the executor container
    securityContext: {}

  server:
    enabled: true
    # only updates base url of resources on client side,
    # it's expected that a proxy server rewrites the request URL and gets rid of this prefix
    # https://github.com/argoproj/argo-workflows/issues/716#issuecomment-433213190
    baseHref: /
    image:
      registry: quay.io
      repository: argoproj/argocli
      # Overrides the image tag whose default is the chart appVersion.
      tag: ""
    # optional map of annotations to be applied to the ui Pods
    podAnnotations: {}
    # Optional labels to add to the UI pods
    podLabels: {}
    # SecurityContext to set on the server pods
    podSecurityContext: {}
    securityContext:
      readOnlyRootFilesystem: false
      runAsNonRoot: true
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
    name: server
    serviceType: ClusterIP
    servicePort: 2746
    # servicePortName: http
    serviceAccount:
      create: true
      name: ""
      annotations: {}
    # Annotations to be applied to the UI Service
    serviceAnnotations: {}
    # Optional labels to add to the UI Service
    serviceLabels: {}
    # Static IP address to assign to loadBalancer
    # service type `LoadBalancer`
    loadBalancerIP: ""
    # Source ranges to allow access to service from. Only applies to
    # service type `LoadBalancer`
    loadBalancerSourceRanges: []
    resources: {}
    replicas: 1
    pdb:
      enabled: false
      # minAvailable: 1
      # maxUnavailable: 1
    ## Node selectors and tolerations for server scheduling to nodes with taints
    ## Ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
    ##
    nodeSelector:
      kubernetes.io/os: linux
    tolerations: []
    affinity: {}
    # Leverage a PriorityClass to ensure your pods survive resource shortages
    # ref: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
    # PriorityClass: system-cluster-critical
    priorityClassName: ""

    # Run the argo server in "secure" mode. Configure this value instead of
    # "--secure" in extraArgs. See the following documentation for more details
    # on secure mode:
    # https://argoproj.github.io/argo-workflows/tls/
    secure: false

    ## Extra environment variables to provide to the argo-server container
    ## extraEnv:
    ##   - name: FOO
    ##     value: "bar"
    extraEnv: []

    # Extra arguments to provide to the Argo server binary.
    extraArgs: []

    ## Additional volumes to the server main container.
    volumeMounts: []
    volumes: []

    ## Ingress configuration.
    ## ref: https://kubernetes.io/docs/user-guide/ingress/
    ##
    ingress:
      enabled: false
      annotations: {}
      labels: {}
      ingressClassName: ""

      ## Argo Workflows Server Ingress.
      ## Hostnames must be provided if Ingress is enabled.
      ## Secrets must be manually created in the namespace
      ##
      hosts:
        []
        # - argocd.example.com
      paths:
        - /
      extraPaths:
        []
        # - path: /*
        #   backend:
        #     serviceName: ssl-redirect
        #     servicePort: use-annotation
        ## for Kubernetes >=1.19 (when "networking.k8s.io/v1" is used)
        # - path: /*
        #   pathType: Prefix
        #   backend:
        #     service
        #       name: ssl-redirect
        #       port:
        #         name: use-annotation
      tls:
        []
        # - secretName: argocd-example-tls
        #   hosts:
        #     - argocd.example.com
      https: false

    clusterWorkflowTemplates:
      # Create a ClusterRole and CRB for the server to access ClusterWorkflowTemplates.
      enabled: true
      # Give the server permissions to edit ClusterWorkflowTemplates.
      enableEditing: true
    sso:
      ## SSO configuration when SSO is specified as a server auth mode.
      ## All the values are required. SSO is activated by adding --auth-mode=sso
      ## to the server command line.
      #
      ## The root URL of the OIDC identity provider.
      # issuer: https://accounts.google.com
      ## Name of a secret and a key in it to retrieve the app OIDC client ID from.
      # clientId:
      #   name: argo-server-sso
      #   key: client-id
      ## Name of a secret and a key in it to retrieve the app OIDC client secret from.
      # clientSecret:
      #   name: argo-server-sso
      #   key: client-secret
      ## The OIDC redirect URL. Should be in the form <argo-root-url>/oauth2/callback.
      # redirectUrl: https://argo/oauth2/callback
      # rbac:
      #   enabled: true
      ## When present, restricts secrets the server can read to a given list.
      ## You can use it to restrict the server to only be able to access the
      ## service account token secrets that are associated with service accounts
      ## used for authorization.
      #   secretWhitelist: []
      ## Scopes requested from the SSO ID provider.  The 'groups' scope requests
      ## group membership information, which is usually used for authorization
      ## decisions.
      # scopes:
      # - groups

  # Influences the creation of the ConfigMap for the workflow-controller itself.
  useDefaultArtifactRepo: false
  useStaticCredentials: true
  artifactRepository:
    # archiveLogs will archive the main container logs as an artifact
    archiveLogs: false
    s3:
      # Note the `key` attribute is not the actual secret, it's the PATH to
      # the contents in the associated secret, as defined by the `name` attribute.
      accessKeySecret:
        # name: <releaseName>-minio
        key: accesskey
      secretKeySecret:
        # name: <releaseName>-minio
        key: secretkey
      insecure: true
      # bucket:
      # endpoint:
      # region:
      # roleARN:
      # useSDKCreds: true
    # gcs:
    # bucket: <project>-argo
    # keyFormat: "{{workflow.namespace}}/{{workflow.name}}/"
    # serviceAccountKeySecret is a secret selector.
    # It references the k8s secret named 'my-gcs-credentials'.
    # This secret is expected to have have the key 'serviceAccountKey',
    # containing the base64 encoded credentials
    # to the bucket.
    #
    # If it's running on GKE and Workload Identity is used,
    # serviceAccountKeySecret is not needed.
    # serviceAccountKeySecret:
    # name: my-gcs-credentials
    # key: serviceAccountKey

helm-controller:
  # Default values for helm-controller.
  # This is a YAML-formatted file.
  # Declare variables to be passed into your templates.
  concurrent: 5

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
