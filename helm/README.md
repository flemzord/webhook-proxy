# Helm Charts for Webhook Proxy

This directory contains Helm charts for deploying Webhook Proxy on Kubernetes.

## Structure

```
helm/
├── README.md                 # This file
└── webhook-proxy/            # Helm Chart for Webhook Proxy
    ├── Chart.yaml            # Chart metadata
    ├── values.yaml           # Default values
    ├── templates/            # Kubernetes templates
    │   ├── _helpers.tpl      # Helper functions
    │   ├── deployment.yaml   # Deployment
    │   ├── service.yaml      # Service
    │   ├── ingress.yaml      # Ingress
    │   ├── configmap.yaml    # ConfigMap
    │   ├── serviceaccount.yaml # Service account
    │   ├── hpa.yaml          # HorizontalPodAutoscaler
    │   └── NOTES.txt         # Installation notes
    ├── examples/             # Configuration examples
    │   ├── values-development.yaml # Development configuration
    │   └── values-production.yaml  # Production configuration
    └── .helmignore           # Files to ignore when packaging
```

## Installation

### Prerequisites

- Kubernetes 1.12+
- Helm 3.0+

### Chart Installation

To install the chart with the release name `my-webhook-proxy`:

```bash
helm install my-webhook-proxy ./webhook-proxy
```

### Installation with Custom Values

To install the chart with custom values:

```bash
helm install my-webhook-proxy ./webhook-proxy -f webhook-proxy/examples/values-production.yaml
```

Or by directly specifying values:

```bash
helm install my-webhook-proxy ./webhook-proxy --set replicaCount=2
```

### Chart Update

To update the configuration:

```bash
helm upgrade my-webhook-proxy ./webhook-proxy -f webhook-proxy/examples/values-production.yaml
```

### Chart Uninstallation

To uninstall the chart:

```bash
helm uninstall my-webhook-proxy
```

## Configuration Examples

The `webhook-proxy/examples/` directory contains configuration examples for different environments:

- `values-development.yaml`: Configuration for a development environment
- `values-production.yaml`: Configuration for a production environment

## Documentation

For more information on chart configuration, see the [chart README](webhook-proxy/README.md). 