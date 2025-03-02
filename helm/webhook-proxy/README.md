# Webhook Proxy Helm Chart

This Helm chart deploys the Webhook Proxy application on a Kubernetes cluster.

## Introduction

Webhook Proxy is a simple application that allows redirecting webhooks to multiple destinations. It is useful for distributing webhook events to different services.

## Prerequisites

- Kubernetes 1.12+
- Helm 3.0+

## Installation

To install the chart with the release name `my-webhook-proxy`:

```bash
helm install my-webhook-proxy ./helm/webhook-proxy
```

## Configuration

The following table lists the configurable parameters of the Webhook Proxy chart and their default values.

| Parameter | Description | Default Value |
|-----------|-------------|-------------------|
| `replicaCount` | Number of deployment replicas | `1` |
| `image.repository` | Docker image to use | `ghcr.io/flemzord/webhook-proxy` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag to use | `""` (uses chart version) |
| `imagePullSecrets` | Secrets for pulling the image | `[]` |
| `nameOverride` | Partially overrides the chart name | `""` |
| `fullnameOverride` | Completely overrides the chart name | `""` |
| `serviceAccount.create` | Specifies whether a service account should be created | `true` |
| `serviceAccount.annotations` | Annotations to add to the service account | `{}` |
| `serviceAccount.name` | Name of the service account to use | `""` |
| `podAnnotations` | Annotations to add to pods | `{}` |
| `podSecurityContext` | Security context for pods | `{}` |
| `securityContext` | Security context for containers | `{}` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `ingress.enabled` | Enables ingress | `false` |
| `ingress.className` | Ingress class to use | `""` |
| `ingress.annotations` | Annotations for ingress | `{}` |
| `ingress.hosts` | Hosts for ingress | `[{"host": "chart-example.local", "paths": [{"path": "/", "pathType": "ImplementationSpecific"}]}]` |
| `ingress.tls` | TLS configuration for ingress | `[]` |
| `resources` | CPU/memory resources | `{}` |
| `autoscaling.enabled` | Enables autoscaling | `false` |
| `autoscaling.minReplicas` | Minimum number of replicas | `1` |
| `autoscaling.maxReplicas` | Maximum number of replicas | `100` |
| `autoscaling.targetCPUUtilizationPercentage` | CPU utilization target for autoscaling | `80` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity | `{}` |
| `config.server.host` | Server host | `"0.0.0.0"` |
| `config.server.port` | Server port | `8080` |
| `config.logging.level` | Logging level | `"info"` |
| `config.logging.format` | Logging format | `"json"` |
| `config.logging.output` | Logging destination | `"stdout"` |
| `config.logging.file_path` | Log file path | `""` |
| `config.endpoints` | Endpoints configuration | `[]` |

### Endpoints Configuration

To configure endpoints, you can use the following format in your `values.yaml` file:

```yaml
config:
  endpoints:
    - path: "/webhook/github"
      destinations:
        - url: "https://example.com/github-webhook"
          headers:
            X-Custom-Header: "custom-value"
        - url: "https://backup-service.example.com/github-events"
```

## Usage Example

Here is an example `values.yaml` file to deploy Webhook Proxy with a custom configuration:

```yaml
replicaCount: 2

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: "nginx"
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: webhook.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: webhook-tls
      hosts:
        - webhook.example.com

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

config:
  server:
    host: "0.0.0.0"
    port: 8080
  
  logging:
    level: "info"
    format: "json"
    output: "stdout"
  
  endpoints:
    - path: "/webhook/github"
      destinations:
        - url: "https://example.com/github-webhook"
          headers:
            X-Custom-Header: "custom-value"
        - url: "https://backup-service.example.com/github-events"
    
    - path: "/webhook/stripe"
      destinations:
        - url: "https://payment-processor.example.com/stripe-events"
        - url: "https://analytics.example.com/payment-events"
          headers:
            Authorization: "Bearer your-token-here"
            Content-Type: "application/json"
```

To install the chart with this values file:

```bash
helm install my-webhook-proxy ./helm/webhook-proxy -f values.yaml
```

## Helm Tests

This chart includes several Helm tests that can be run to verify that the deployment is working correctly. The tests check:

1. **Connectivity** - Verifies that the service is accessible
2. **Service** - Verifies that the service is correctly configured and responds to requests
3. **Configuration** - Verifies that the ConfigMap contains the required configuration sections
4. **Default Values** - Verifies that the chart's default values are correctly applied
5. **Endpoints** - Verifies that the configured endpoints are correctly defined in the ConfigMap

To run the tests after installing the chart:

```bash
helm test my-webhook-proxy
```

Example of successful output:

```
NAME: my-webhook-proxy
LAST DEPLOYED: Mon Jan 01 2024 12:00:00
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE:     my-webhook-proxy-test-connection
Last Started:   Mon Jan 01 2024 12:05:00
Last Completed: Mon Jan 01 2024 12:05:05
Phase:          Succeeded
TEST SUITE:     my-webhook-proxy-test-service
Last Started:   Mon Jan 01 2024 12:05:05
Last Completed: Mon Jan 01 2024 12:05:10
Phase:          Succeeded
TEST SUITE:     my-webhook-proxy-test-config
Last Started:   Mon Jan 01 2024 12:05:10
Last Completed: Mon Jan 01 2024 12:05:15
Phase:          Succeeded
TEST SUITE:     my-webhook-proxy-test-default-values
Last Started:   Mon Jan 01 2024 12:05:15
Last Completed: Mon Jan 01 2024 12:05:20
Phase:          Succeeded
TEST SUITE:     my-webhook-proxy-test-endpoints
Last Started:   Mon Jan 01 2024 12:05:20
Last Completed: Mon Jan 01 2024 12:05:25
Phase:          Succeeded
```

These tests are useful for verifying that the chart has been correctly installed and that the configuration is valid. 