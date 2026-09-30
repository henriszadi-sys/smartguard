# SmartGUARD

Module installé sur un serveur qui suit la date de fin d'un contrat de support lié à un logiciel, avertit les utilisateurs à l'approche de l'échéance, puis exécute des actions à l'échéance ou à une date d'arrêt planifiée (arrêt de services, blocage d'adresses, scripts). Anciennement nommé **LicGuard** : ne plus utiliser ce nom dans le code, les fichiers ni les messages.

La spécification fonctionnelle de référence est `Cahier_des_charges_SmartGUARD.md` (v1.3). En cas de doute, elle fait foi ; les décisions de la section 15 sont des propositions à valider : les suivre par défaut mais les isoler pour pouvoir les changer facilement.

## Stack

- **Langage** : Go (dernière version stable), un seul binaire sans dépendance pour Windows Server et Linux.
- **Service natif** : `golang.org/x/sys/windows/svc` sous Windows, unité systemd sous Linux.
- **Proxy inverse** : `net/http/httputil` (mode automatique, injection du bandeau dans les pages HTML).
- **Interface web** (assistant + administration) : HTML/JS simple embarqué avec `embed`, sans étape de build front obligatoire.
- **Stockage** : un fichier de configuration par module (JSON ou SQLite), sans serveur de base de données.
- **Mot de passe** : hachage bcrypt ou argon2, jamais en clair.

## Structure proposée

```
cmd/smartguard/      point d'entrée (service, CLI : status, check, passwd)
internal/config/     lecture, écriture, export/import de la configuration
internal/scheduler/  échéances, rappel J-x, arrêt planifié, recalcul au démarrage
internal/actions/    arrêt de services, blocage d'adresses, exécution de scripts
internal/proxy/      mode automatique, injection du message, page « accès suspendu »
internal/license/    licence par poste, empreinte, activation, transfert
internal/admin/      interface web d'administration, authentification, verrouillage
internal/wizard/     assistant d'installation en six étapes
internal/logging/    installation.log et config.log
web/                 fichiers statiques embarqués
docs/                cahier des charges et documentation technique
```

## Commandes

```bash
go build ./cmd/smartguard          # build local
go test ./...                      # tests
go vet ./... && gofmt -l .         # vérifications
GOOS=windows GOARCH=amd64 go build -o dist/smartguard.exe ./cmd/smartguard
GOOS=linux   GOARCH=amd64 go build -o dist/smartguard     ./cmd/smartguard
```

Adapter ces commandes si la structure du dépôt change.

## Règles fonctionnelles à ne jamais casser

- L'échéance tombe à **00:00 le jour de la date de fin**, dans le fuseau configuré du serveur.
- Le recul de l'horloge du serveur ne doit jamais repousser l'échéance : conserver un dernier horodatage connu et ne jamais reculer.
- Au démarrage du service, recalculer l'état et exécuter **une seule fois** les actions manquées, puis journaliser.
- La date d'arrêt planifiée et les actions de fin de contrat sont indépendantes ; sans date d'arrêt, aucune action.
- Le rappel : orange puis rouge à partir de **J-7**, texte par défaut du cahier des charges, nombre de jours restants affiché.
- Renouvellement : nouvelle date de fin saisie dans l'administration, puis action explicite « Réactiver les services ». **Ne jamais réactiver automatiquement.**
- Plusieurs modules par serveur : un service, un port et un dossier de configuration distincts par module. Renommer un module renomme le service.
- Désinstallation : supprime le service, la règle de pare-feu et les fichiers du module.
- Licence : une licence ne peut être active que sur un poste à la fois ; transfert par désactivation puis activation.

## Sécurité

- L'assistant d'installation n'est accessible que par le lien secret affiché par le programme.
- Identifiant d'administration `admin` ; blocage après **10 échecs** de mot de passe.
- Scripts : exécutés uniquement depuis l'administration authentifiée, délai maximum, 3 tentatives, échec journalisé.
- HTTPS pour l'administration (certificat auto-signé par défaut, remplaçable).
- SmartGUARD est un outil de rappel contractuel, **pas une protection anti-piratage** : ne pas promettre le contraire dans l'interface ni la documentation.
- Ne jamais écrire de secrets (mots de passe, clés de licence) dans les journaux.

## Journaux

- `installation.log` : installations et mises à jour.
- `config.log` : démarrage, erreurs, actions à l'expiration ou à l'arrêt planifié.
- Chaque action d'échéance ou d'arrêt est journalisée avec son résultat.

## Conventions

- Langue de l'interface utilisateur et de la documentation : **français**. Identifiants de code et commentaires : anglais ou français, mais cohérents dans tout le dépôt.
- Petits commits, un sujet par commit, message à l'impératif.
- Tout comportement lié aux dates (échéance, J-7, changement d'heure, recul d'horloge, redémarrage) doit avoir un test avec une horloge injectable.
- Ne pas ajouter de dépendance sans nécessité : le binaire doit rester unique et léger, et éviter les faux positifs d'antivirus (pas d'empaquetage ni d'obfuscation).
- Avant de modifier le comportement décrit dans le cahier des charges, mettre à jour ce dernier et son historique de versions.

## Points encore ouverts

Voir la section 15 du cahier des charges : chaque décision proposée reste à valider par le commanditaire, en particulier la définition d'un « poste » et le mode hors ligne de la licence.
