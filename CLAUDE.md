# SmartGUARD

Module installé sur un serveur qui suit la date de fin d'un contrat de support lié à un logiciel, avertit les utilisateurs à l'approche de l'échéance, puis exécute des actions à l'échéance ou à une date d'arrêt planifiée (arrêt de services, blocage d'adresses, scripts). Anciennement nommé **LicGuard** : ne plus utiliser ce nom dans le code, les fichiers ni les messages. Seules exceptions, pour la mise à jour des installations existantes : la reprise de l'ancien registre (`platform.LegacyRegistryDir`) et le chemin d'administration `/_licguard` que les modules déjà installés conservent dans leur configuration.

La spécification fonctionnelle de référence est `Cahier_des_charges_SmartGUARD.md` (v1.3). En cas de doute, elle fait foi ; les décisions de la section 15 sont des propositions à valider : les suivre par défaut mais les isoler pour pouvoir les changer facilement.

## Stack

- **Langage** : Go (dernière version stable ; `go.mod` exige 1.24 minimum), un seul binaire sans dépendance pour Windows Server et Linux.
- **Service natif** : `github.com/kardianos/service` (service Windows, unité systemd sous Linux), avec `golang.org/x/sys` pour les droits administrateur sous Windows.
- **Proxy inverse** : `net/http/httputil` (mode automatique, injection du bandeau dans les pages HTML).
- **Interface web** (assistant + administration) : HTML/JS simple embarqué avec `embed` (paquet `web`), sans étape de build front.
- **Stockage** : par module, un `config.json` (paramètres) et un `config.state.json` (dernière heure vue, actions exécutées), sans serveur de base de données. Registre des modules installés : `%ProgramData%\SmartGUARD\installations.json` ou `/etc/smartguard/installations.json`.
- **Mot de passe** : PBKDF2-SHA256 de la bibliothèque standard (210 000 itérations, sel aléatoire), jamais en clair.

## Structure

```
cmd/smartguard/      point d'entrée : assistant, service, CLI (status, check, set-password)
internal/config/     configuration, état persistant, validation, hachage du mot de passe
internal/scheduler/  décompte, rappel J-x, échéance, anti-recul d'horloge, Watcher (actions une seule fois)
internal/actions/    arrêt/réactivation des services, scripts (3 tentatives, délai maximum)
internal/proxy/      mode automatique, injection du bandeau, page « accès suspendu »
internal/admin/      routes du module, connexion, sessions, verrouillage, API d'administration
internal/wizard/     assistant d'installation en six étapes, registre des modules
internal/platform/   spécificités Windows / Linux (droits, pare-feu, services, raccourcis)
internal/logging/    config.log et installation.log
internal/version/    numéro de version
web/                 pages et scripts embarqués
Windows/, Linux/     exécutables livrés (SmartGUARD-Setup.exe, smartguard-setup)
```

Pas encore implémenté : `internal/license/` (licence par poste), en attente de la définition d'un « poste » (section 15 du cahier des charges).

L'horloge est injectable (`scheduler.Watcher.Now`, `admin.Server.Now`, paramètre `now` de `scheduler.ComputeStatus`) : l'utiliser dans les tests plutôt que `time.Now`.

## Commandes

```bash
go build ./cmd/smartguard          # build local
go test ./...                      # tests
go vet ./... && gofmt -l .         # vérifications (gofmt ne doit rien lister)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o Windows/SmartGUARD-Setup.exe ./cmd/smartguard
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o Linux/smartguard-setup ./cmd/smartguard
```

Passer `internal/version` et le titre de `LISEZMOI.md` à la nouvelle version avant de livrer.

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
