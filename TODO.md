# Plan d'action pour le développement du webhook-proxy

## Description du projet
Un service proxy pour recevoir des webhooks et les transférer à plusieurs destinations configurées, avec une journalisation détaillée de chaque étape.

## Composants principaux
- Serveur HTTP pour recevoir les webhooks
- Gestionnaire de proxy pour transférer les webhooks
- Système de configuration pour définir les endpoints et les destinations
- Système de journalisation pour enregistrer les événements

## Structure du fichier de configuration YAML
```yaml
server:
  port: 8080
  host: "0.0.0.0"

logging:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: "/path/to/log/file.log"

endpoints:
  - path: "/webhook/github"
    destinations:
      - url: "https://destination1.example.com/webhook"
        headers:
          Content-Type: "application/json"
          X-Custom-Header: "custom-value"
```

## Architecture du code
- `cmd/webhook-proxy/main.go`: Point d'entrée de l'application
- `config/`: Gestion de la configuration
- `logger/`: Système de journalisation
- `server/`: Serveur HTTP
- `proxy/`: Gestionnaire de proxy pour transférer les webhooks

## Dépendances externes
- HTTP Framework: Chi
- Logging: logrus
- Configuration: viper ou yaml.v3

## Plan de développement

### Phase 1: Configuration de base et serveur HTTP ✅
- [x] Créer la structure du projet
- [x] Configurer le système de journalisation
- [x] Implémenter le serveur HTTP de base
- [x] Créer le gestionnaire de proxy simple
- [x] Remplacer Gin par Chi

### Phase 2: Configuration et validation ✅
- [x] Implémenter le parser de configuration YAML
- [x] Créer les structures de données pour représenter la configuration
- [x] Ajouter la validation de la configuration
- [x] Permettre le chargement de la configuration depuis des variables d'environnement
- [x] Ajouter des tests unitaires pour la configuration
- [x] Créer un fichier d'exemple de configuration

### Phase 3: Amélioration du proxy et gestion des erreurs ✅
- [x] Ajouter un mécanisme de retry pour les destinations en échec
- [x] Implémenter la gestion des timeouts
- [x] Ajouter des métriques pour surveiller les performances
- [x] Améliorer la journalisation des erreurs
- [x] Ajouter des tests unitaires pour le proxy
- [x] Ajouter des endpoints pour les métriques et la santé

### Phase 4: Documentation et API ✅
- [x] Créer un fichier openapi.yaml a la racine du repository pour documenter l'API

### Phase 5: Qualité du code et outillage
- [ ] Ajouter golangci-lint pour le linting du code
- [ ] Configurer les règles de linting adaptées au projet
- [ ] Corriger les problèmes de linting existants

### Phase 6: Déploiement et distribution
- [ ] Configurer GoReleaser pour la création des binaires
- [ ] Créer un Dockerfile pour la conteneurisation

### Phase 7: Intégration continue
- [ ] Configurer GitHub Actions pour l'intégration continue
- [ ] Ajouter des workflows pour les tests automatiques
- [ ] Configurer le linting automatique
- [ ] Ajouter la publication automatique des releases avec GoReleaser
- [ ] Configurer la construction et la publication des images Docker