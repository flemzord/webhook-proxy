# Webhook Proxy

Un service proxy pour recevoir des webhooks et les transférer à plusieurs destinations.

## Fonctionnalités

- Réception de webhooks sur des endpoints configurables
- Transfert des webhooks à plusieurs destinations
- Journalisation détaillée des requêtes et des réponses
- Configuration via fichier YAML ou variables d'environnement
- Validation de la configuration
- Mécanisme de retry pour les destinations en échec
- Métriques pour surveiller les performances
- Endpoints de santé et de métriques

## Installation

```bash
# Cloner le dépôt
git clone https://github.com/flemzord/webhook-proxy.git
cd webhook-proxy

# Compiler le projet
go build -o webhook-proxy ./cmd/webhook-proxy

# Exécuter le service
./webhook-proxy -config config.yaml
```

## Configuration

### Fichier YAML

Créez un fichier de configuration YAML basé sur l'exemple fourni (`config.example.yaml`) :

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 8080

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: ""

# Endpoints configuration
endpoints:
  - path: "/webhook/github"
    destinations:
      - url: "https://example.com/github-webhook"
        headers:
          X-Custom-Header: "custom-value"
      - url: "https://backup-service.example.com/github-events"
```

### Variables d'environnement

Vous pouvez également configurer le service à l'aide de variables d'environnement, qui ont priorité sur les valeurs du fichier YAML :

| Variable | Description | Exemple |
|----------|-------------|---------|
| `WEBHOOK_PROXY_SERVER_HOST` | Hôte du serveur | `0.0.0.0` |
| `WEBHOOK_PROXY_SERVER_PORT` | Port du serveur | `8080` |
| `WEBHOOK_PROXY_LOG_LEVEL` | Niveau de journalisation (debug, info, warn, error) | `info` |
| `WEBHOOK_PROXY_LOG_FORMAT` | Format de journalisation (json, text) | `json` |
| `WEBHOOK_PROXY_LOG_OUTPUT` | Destination de journalisation (stdout, stderr, file) | `stdout` |
| `WEBHOOK_PROXY_LOG_FILE_PATH` | Chemin du fichier de journalisation (requis si output=file) | `/var/log/webhook-proxy.log` |

**Note**: Les endpoints doivent être configurés via le fichier YAML.

## Utilisation

1. Démarrez le service avec votre fichier de configuration :
   ```bash
   ./webhook-proxy -config config.yaml
   ```

2. Envoyez des webhooks aux endpoints configurés, par exemple :
   ```bash
   curl -X POST http://localhost:8080/webhook/github -d '{"event":"push","repository":"example"}'
   ```

3. Le service transférera la requête à toutes les destinations configurées pour cet endpoint.

## Endpoints système

En plus des endpoints configurés pour les webhooks, le service expose les endpoints système suivants :

### Métriques

- **GET /metrics** : Renvoie les métriques du service au format JSON, incluant :
  - Nombre total de requêtes
  - Nombre de requêtes réussies
  - Nombre de requêtes échouées
  - Nombre de retries
  - Taux de succès
  - Métriques par destination

- **POST /metrics/reset** : Réinitialise toutes les métriques

### Santé

- **GET /health** : Renvoie l'état de santé du service

Exemple de réponse de l'endpoint `/metrics` :
```json
{
  "global": {
    "total_requests": 42,
    "successful_requests": 40,
    "failed_requests": 2,
    "retries": 1,
    "success_rate": 95.23
  },
  "endpoints": {
    "/webhook/github": {
      "total_requests": 42,
      "successful_requests": 40,
      "failed_requests": 2,
      "retries": 1,
      "avg_response_time_ms": 125.5,
      "status_codes": {
        "200": 40,
        "500": 2
      },
      "destinations": {
        "https://example.com/github-webhook": {
          "total_requests": 21,
          "successful_requests": 20,
          "failed_requests": 1,
          "retries": 0,
          "avg_response_time_ms": 100.2
        },
        "https://backup-service.example.com/github-events": {
          "total_requests": 21,
          "successful_requests": 20,
          "failed_requests": 1,
          "retries": 1,
          "avg_response_time_ms": 150.8,
          "last_error": "connection timeout",
          "last_error_time": "2023-01-01T12:00:00Z"
        }
      }
    }
  },
  "timestamp": "2023-01-01T12:30:00Z"
}
```

## Développement

Pour plus de détails sur le développement et les fonctionnalités à venir, consultez le fichier [TODO.md](TODO.md).

## Licence

Ce projet est sous licence MIT. Voir le fichier LICENSE pour plus de détails.
