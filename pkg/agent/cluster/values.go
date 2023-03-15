package cluster

const (
	// format by. featureGates, repository, tag, limits.cpu, limits.memory, requests.cpu, requests.memory
	kruiseValues = `
crds:
  managed: true
installation:
  namespace: kruise-system
  roleListGroups:
    - '*'
featureGates: "%s"
manager:
  log:
    level: "4"
  replicas: 2
  image:
    repository: %s
    tag: %s
  webhook:
    port: 9876
  metrics:
    port: 8080
  healthProbe:
    port: 8000
  resyncPeriod: "0"
  resources:
    limits:
      cpu: %s
      memory: %s
    requests:
      cpu: %s
      memory: %s
  hostNetwork: false
  nodeAffinity: {}
  nodeSelector: {}
  tolerations: []
webhookConfiguration:
  failurePolicy:
    pods: Ignore
  timeoutSeconds: 30
daemon:
  log:
    level: "4"
  port: 10221
  socketLocation: "/var/run"
  nodeSelector: {}
  resources:
    limits:
      cpu: 50m
      memory: 128Mi
    requests:
      cpu: "0"
      memory: "0"
`
	// format by. keda.repository, keda.tag, metricsApiServer.repository, metricsApiServer.tag
	kedaValues = `image:
  keda:
    repository: %s
    tag: %s
  metricsApiServer:
    repository: %s
    tag: %s
  pullPolicy: Always
crds:
  install: true
watchNamespace: ""
operator:
  name: keda-operator
  replicaCount: 1
rbac:
  create: true
serviceAccount:
  create: true
  name: keda-operator
service:
  type: ClusterIP
  portHttp: 80
  portHttpTarget: 8080
  portHttps: 443
  portHttpsTarget: 6443
resources:
  operator:
    limits:
      cpu: 1
      memory: 1000Mi
    requests:
      cpu: 100m
      memory: 100Mi
  metricServer:
    limits:
      cpu: 1
      memory: 1000Mi
    requests:
      cpu: 100m
      memory: 100Mi`
)
