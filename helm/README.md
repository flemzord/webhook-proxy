# Helm Charts pour Webhook Proxy

Ce répertoire contient les charts Helm pour déployer Webhook Proxy sur Kubernetes.

## Structure

```
helm/
├── README.md                 # Ce fichier
└── webhook-proxy/            # Chart Helm pour Webhook Proxy
    ├── Chart.yaml            # Métadonnées du chart
    ├── values.yaml           # Valeurs par défaut
    ├── templates/            # Templates Kubernetes
    │   ├── _helpers.tpl      # Fonctions d'aide
    │   ├── deployment.yaml   # Déploiement
    │   ├── service.yaml      # Service
    │   ├── ingress.yaml      # Ingress
    │   ├── configmap.yaml    # ConfigMap
    │   ├── serviceaccount.yaml # Compte de service
    │   ├── hpa.yaml          # HorizontalPodAutoscaler
    │   └── NOTES.txt         # Notes d'installation
    ├── examples/             # Exemples de configuration
    │   ├── values-development.yaml # Configuration pour développement
    │   └── values-production.yaml  # Configuration pour production
    └── .helmignore           # Fichiers à ignorer lors du packaging
```

## Installation

### Prérequis

- Kubernetes 1.12+
- Helm 3.0+

### Installation du chart

Pour installer le chart avec le nom de release `my-webhook-proxy` :

```bash
helm install my-webhook-proxy ./webhook-proxy
```

### Installation avec des valeurs personnalisées

Pour installer le chart avec des valeurs personnalisées :

```bash
helm install my-webhook-proxy ./webhook-proxy -f webhook-proxy/examples/values-production.yaml
```

Ou en spécifiant directement les valeurs :

```bash
helm install my-webhook-proxy ./webhook-proxy --set replicaCount=2
```

### Mise à jour du chart

Pour mettre à jour la configuration :

```bash
helm upgrade my-webhook-proxy ./webhook-proxy -f webhook-proxy/examples/values-production.yaml
```

### Désinstallation du chart

Pour désinstaller le chart :

```bash
helm uninstall my-webhook-proxy
```

## Exemples de configuration

Le répertoire `webhook-proxy/examples/` contient des exemples de configuration pour différents environnements :

- `values-development.yaml` : Configuration pour un environnement de développement
- `values-production.yaml` : Configuration pour un environnement de production

## Documentation

Pour plus d'informations sur la configuration du chart, consultez le [README du chart](webhook-proxy/README.md). 