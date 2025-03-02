# Webhook Proxy Helm Chart

Ce chart Helm déploie l'application Webhook Proxy sur un cluster Kubernetes.

## Introduction

Webhook Proxy est une application simple qui permet de rediriger des webhooks vers plusieurs destinations. Elle est utile pour distribuer des événements de webhook à différents services.

## Prérequis

- Kubernetes 1.12+
- Helm 3.0+

## Installation

Pour installer le chart avec le nom de release `my-webhook-proxy` :

```bash
helm install my-webhook-proxy ./helm/webhook-proxy
```

## Configuration

Le tableau suivant liste les paramètres configurables du chart Webhook Proxy et leurs valeurs par défaut.

| Paramètre | Description | Valeur par défaut |
|-----------|-------------|-------------------|
| `replicaCount` | Nombre de réplicas du déploiement | `1` |
| `image.repository` | Image Docker à utiliser | `ghcr.io/flemzord/webhook-proxy` |
| `image.pullPolicy` | Politique de pull de l'image | `IfNotPresent` |
| `image.tag` | Tag de l'image à utiliser | `""` (utilise la version du chart) |
| `imagePullSecrets` | Secrets pour pull l'image | `[]` |
| `nameOverride` | Remplace partiellement le nom du chart | `""` |
| `fullnameOverride` | Remplace complètement le nom du chart | `""` |
| `serviceAccount.create` | Spécifie si un compte de service doit être créé | `true` |
| `serviceAccount.annotations` | Annotations à ajouter au compte de service | `{}` |
| `serviceAccount.name` | Nom du compte de service à utiliser | `""` |
| `podAnnotations` | Annotations à ajouter aux pods | `{}` |
| `podSecurityContext` | Contexte de sécurité pour les pods | `{}` |
| `securityContext` | Contexte de sécurité pour les conteneurs | `{}` |
| `service.type` | Type de service Kubernetes | `ClusterIP` |
| `service.port` | Port du service | `8080` |
| `ingress.enabled` | Active l'ingress | `false` |
| `ingress.className` | Classe d'ingress à utiliser | `""` |
| `ingress.annotations` | Annotations pour l'ingress | `{}` |
| `ingress.hosts` | Hôtes pour l'ingress | `[{"host": "chart-example.local", "paths": [{"path": "/", "pathType": "ImplementationSpecific"}]}]` |
| `ingress.tls` | Configuration TLS pour l'ingress | `[]` |
| `resources` | Ressources CPU/mémoire | `{}` |
| `autoscaling.enabled` | Active l'autoscaling | `false` |
| `autoscaling.minReplicas` | Nombre minimum de réplicas | `1` |
| `autoscaling.maxReplicas` | Nombre maximum de réplicas | `100` |
| `autoscaling.targetCPUUtilizationPercentage` | Cible d'utilisation CPU pour l'autoscaling | `80` |
| `nodeSelector` | Sélecteur de nœuds | `{}` |
| `tolerations` | Tolérances | `[]` |
| `affinity` | Affinité | `{}` |
| `config.server.host` | Hôte du serveur | `"0.0.0.0"` |
| `config.server.port` | Port du serveur | `8080` |
| `config.logging.level` | Niveau de logging | `"info"` |
| `config.logging.format` | Format de logging | `"json"` |
| `config.logging.output` | Destination de logging | `"stdout"` |
| `config.logging.file_path` | Chemin du fichier de log | `""` |
| `config.endpoints` | Configuration des endpoints | `[]` |

### Configuration des endpoints

Pour configurer les endpoints, vous pouvez utiliser le format suivant dans votre fichier `values.yaml` :

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

## Exemple d'utilisation

Voici un exemple de fichier `values.yaml` pour déployer Webhook Proxy avec une configuration personnalisée :

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

Pour installer le chart avec ce fichier de valeurs :

```bash
helm install my-webhook-proxy ./helm/webhook-proxy -f values.yaml
``` 