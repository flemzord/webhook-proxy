# Webhook Proxy

Un service proxy pour recevoir des webhooks et les transférer à plusieurs destinations.

## Fonctionnalités

- Réception de webhooks sur des endpoints configurables
- Transfert des webhooks à plusieurs destinations
- Journalisation détaillée des requêtes et des réponses
- Configuration via fichier YAML ou variables d'environnement
- Validation de la configuration

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

## Développement

Pour plus de détails sur le développement et les fonctionnalités à venir, consultez le fichier [TODO.md](TODO.md).

## Licence

Ce projet est sous licence MIT. Voir le fichier LICENSE pour plus de détails.
