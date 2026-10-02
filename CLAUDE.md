# SmartGUARD

Module installé sur le serveur d'un client qui suit la date de fin d'une **licence** ou d'un **contrat** (support technique, maintenance, abonnement) lié à un logiciel du fournisseur, avertit les utilisateurs à l'approche de l'échéance, puis exécute, à la date de fin et si l'option d'arrêt est cochée, les actions configurées par le fournisseur (arrêt de services, blocage d'adresses, scripts). Un **portail fournisseur en SaaS** (à venir) permet au fournisseur de suivre les échéances de tous ses clients et d'en recevoir les alertes.

Anciennement nommé **LicGuard** : ne plus utiliser ce nom dans le code, les fichiers ni les messages. Seules exceptions, pour la mise à jour des installations existantes : la reprise de l'ancien registre (`platform.LegacyRegistryDir`) et le chemin d'administration `/_licguard` que les modules déjà installés conservent dans leur configuration.

## Documents de référence

- `Cahier_des_charges_SmartGUARD.md` (**v1.20**) : spécification fonctionnelle. Elle fait foi. Les décisions de la section 16 marquées *à valider* sont des propositions : les suivre par défaut mais les isoler (configuration, paramètres) pour pouvoir les changer.
- `docs/Plan_de_developpement_SmartGUARD.md` : découpage en lots, critères d'acceptation couverts, suivi.
- `docs/Etat_des_lieux_depot_SmartGUARD.md` : ce que le code v1.6.0 couvrait au départ et les écarts avec le cahier des charges.
- `EMISSION_CLES.md` : émission des licences SmartGUARD par l'éditeur (ne pas livrer aux clients).

## Composants

- **Module SmartGUARD** (ce dépôt, existant) : installé sur le serveur du client, binaire unique, fonctionne entièrement sans le portail.
- **Portail fournisseur** (à venir, lots 6-7) : SaaS multi-fournisseurs hébergé par l'éditeur, PostgreSQL, API REST.
- **Espace client** (option, lot 8a) : signalement d'incidents et suivi, hébergé dans le SaaS.
- **Application mobile du fournisseur** (option, lot 8b) : Flutter, même API que le portail.

Pas de revendeur. Les paiements entre le fournisseur et son client sont **hors périmètre** : ne développer ni devis, ni facture, ni encaissement entre eux.

## Modèle commercial (impact sur le code)

- **Licence SmartGUARD** : une par serveur de l'application à contrôler, perpétuelle, rattachée au compte du fournisseur, couvrant tous les modules du serveur. **Bloquante à l'installation** (pas de nouvelle installation sans licence valide, contrôle `RequireForInstall` dans l'assistant et la commande `install`) ; la mise à jour d'un module existant est acceptée sans licence et signalée « licence à activer » (`wizard.RequireLicenseOnUpdate = false`, point 39) ; une fois installé, le module ne s'arrête jamais à cause de la licence.
- Droits par niveau (portail) : **licence seule** = accès de base (saisie des clients et échéances, une notification e-mail à J-30 par échéance) ; **abonnement** (Essentiel ≤ 10, Pro 11-50, Entreprise > 50 serveurs rattachés) = portail complet ; **options** (mobile, espace client) activées par compte. Droits contrôlés côté serveur.
- Les prix sont des paramètres, jamais codés en dur.
- Toujours qualifier « licence » : licence SmartGUARD (paquet `internal/license`) vs licence du logiciel du fournisseur (type d'échéance).

## Stack

- **Langage** : Go (`go.mod` exige 1.24 minimum), un seul binaire sans dépendance pour Windows (Windows 10 et Windows Server 2016 minimum) et Linux (systemd).
- **Service natif** : `github.com/kardianos/service` (service Windows, unité systemd), avec `golang.org/x/sys` pour les droits administrateur sous Windows.
- **Proxy inverse** : `net/http/httputil` (mode automatique, injection du bandeau dans les pages HTML).
- **Interface web** (assistant + administration) : HTML/JS embarqué avec `embed` (paquet `web`), sans étape de build front.
- **Stockage** : par module, `config.json` (paramètres), `config.state.json` (dernière heure vue, actions exécutées) sans serveur de base de données. Registre des modules : `%ProgramData%\SmartGUARD\installations.json` ou `/etc/smartguard/installations.json` ; licence SmartGUARD du serveur, unique pour tous les modules : `license.json` dans le même dossier (l'ancien `config.license.json` d'un module est repris automatiquement puis supprimé).
- **Mot de passe** : PBKDF2-SHA256 de la bibliothèque standard (210 000 itérations, sel aléatoire), jamais en clair.
- **Portail** (à venir) : binaire Go séparé, PostgreSQL, API REST, e-mails transactionnels. **Mobile** : Flutter, Firebase Cloud Messaging et APNs. Codes de renouvellement signés Ed25519.

## Structure

Existant :

```
cmd/smartguard/      point d'entrée : assistant, service, CLI (status, check, set-password, license)
cmd/sgkeys/          outil du fournisseur (non livré aux clients) : keygen, issue, verify
internal/config/     configuration, état persistant, validation, hachage du mot de passe
internal/scheduler/  décompte, rappel J-x, échéance, anti-recul d'horloge, Watcher (actions une seule fois)
internal/actions/    arrêt/réactivation des services, scripts (3 tentatives, délai maximum)
internal/proxy/      mode automatique, injection du bandeau, page « accès suspendu »
internal/admin/      routes du module, connexion, sessions, verrouillage, API d'administration
internal/wizard/     assistant d'installation en six étapes, registre des modules
internal/platform/   spécificités Windows / Linux (droits, pare-feu, services, raccourcis)
internal/license/    licence SmartGUARD par serveur : clés signées Ed25519, identifiant machine, liaison locale, public.key embarquée
internal/logging/    config.log et installation.log
internal/version/    numéro de version
web/                 pages et scripts embarqués
docs/                plan de développement, état des lieux
```

À venir (selon le plan) : `internal/portalclient/` (enrôlement, signal), `internal/renewal/` (codes de renouvellement signés), `internal/healthcheck/` (détection de panne), `cmd/smartguard-portal/` et `internal/portal/`, `internal/incidents/`, `mobile/`.

Les exécutables (`Windows/SmartGUARD-Setup.exe`, `Linux/smartguard-setup`) et l'archive de livraison ne sont **pas versionnés** : ils sont publiés comme « releases » GitHub.

## Licence SmartGUARD (`internal/license/`)

- Un « poste » est le serveur (identifiant machine haché) ; liaison locale hors ligne. Sans portail, une même clé utilisée sur deux serveurs n'est pas détectable (seule une installation copiée l'est).
- Clés signées (`SGL1.…`, Ed25519) : l'éditeur garde la clé privée (`sgkeys keygen`, dossier `cles-editeur/` et fichiers `*.sgpriv` ignorés par git, **jamais dans le dépôt**) ; la clé publique est dans `internal/license/public.key`, intégrée au binaire. **Mode signé actif depuis le 2 octobre 2026** (v1.9.0) : les clés `SGRD-…` sont refusées. `public.key` vide = mode non signé (réservé aux tests). Après avoir changé `public.key`, reconstruire les exécutables. Procédure : `EMISSION_CLES.md`.
- Ne jamais afficher ni journaliser la clé complète (`license.Mask`).
- Identifiant machine, vérification et format de clé restent isolés dans ce paquet (contrôle en ligne et révocation via le portail à venir).

## Horloge

L'horloge est injectable (`scheduler.Watcher.Now`, `admin.Server.Now`, paramètre `now` de `scheduler.ComputeStatus`) : l'utiliser dans les tests plutôt que `time.Now`.

## Commandes

```bash
go build ./cmd/smartguard          # build local
go test ./...                      # tests
go vet ./... && gofmt -l .         # vérifications (gofmt ne doit rien lister)
go run ./cmd/sgkeys issue -key fournisseur.sgpriv -customer "Société X"   # émettre une licence (voir EMISSION_CLES.md)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o Windows/SmartGUARD-Setup.exe ./cmd/smartguard
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o Linux/smartguard-setup ./cmd/smartguard
```

Passer `internal/version` et le titre de `LISEZMOI.md` à la nouvelle version avant de livrer.

## Règles fonctionnelles à ne jamais casser

- **Date de fin = date d'arrêt**, pour chaque échéance : à **00:00 le jour de la date de fin**, selon l'heure du serveur du client, le message passe à « expiré ».
- L'arrêt (services, blocage d'adresses, scripts) n'a lieu, **une seule fois**, à cette date que si l'option `stop_on_end` est cochée. Option décochée : jamais d'action.
- `stop_date` n'existe plus : à la lecture d'un ancien `config.json`, sa valeur devient `end_date` et `stop_on_end` passe à vrai.
- Un module suit **plusieurs échéances** (licence, contrat, abonnement), chacune avec sa date, son option d'arrêt, son rappel, son message et ses actions ; une configuration v1.6 à une seule échéance est migrée sans perte (type « contrat de support »). Ne pas coder « contrat » en dur.
- Le recul de l'horloge du serveur ne doit jamais repousser l'échéance : conserver un dernier horodatage connu et ne jamais reculer.
- Au démarrage du service, recalculer l'état et exécuter **une seule fois** les actions d'arrêt manquées, puis journaliser.
- Le rappel : orange puis **rouge à partir de J-7**, texte par défaut selon le type d'échéance, nombre de jours restants affiché.
- Renouvellement : nouvelle date de fin saisie dans l'administration (ou par code signé, à venir), puis action explicite « Réactiver les services ». **Ne jamais réactiver automatiquement.**
- Plusieurs modules par serveur : un service, un port et un dossier de configuration distincts par module. Renommer un module renomme le service.
- Désinstallation : supprime le service, la règle de pare-feu et les fichiers du module.
- Licence SmartGUARD : active sur un seul serveur à la fois ; transfert par désactivation puis activation.
- Réglages réservés au fournisseur (page « accès suspendu », scripts, comportement au redémarrage, sauvegardes) : modifiables uniquement par le compte du technicien du fournisseur, jamais par un accès client ni à distance depuis le portail (lot 5).
- Les arrêts et blocages chez le client ne viennent **que** des actions configurées par le fournisseur. Aucun code de l'éditeur (portail, licence SmartGUARD, abonnement) ne doit pouvoir déclencher un arrêt ou un blocage chez le client, même en cas d'impayé du fournisseur.
- Le module ne dépend **jamais** du portail pour les rappels, échéances et actions ; échanges avec le portail en HTTPS sortant uniquement.

## Règles du portail et de l'espace client (à venir)

- Isolation stricte entre fournisseurs et entre clients ; tester ce cloisonnement sur chaque requête.
- L'administration de l'éditeur n'affiche que les identifiants des clients des fournisseurs ; accès techniques de l'éditeur journalisés.
- Le fournisseur n'est **jamais alerté d'un incident sans action du client** ; une panne détectée et non confirmée est seulement visible dans le portail.
- Détection de panne désactivée par défaut, seuil de 5 minutes modifiable ; un arrêt provoqué par SmartGUARD n'est jamais une panne.
- L'alerte « module silencieux » n'est pas un incident : envoyée au fournisseur sans confirmation du client.
- Le signalement est enregistré dans le SaaS et ne dépend jamais du serveur du client.

## Sécurité

- L'assistant d'installation n'est accessible que par le lien secret affiché par le programme.
- Identifiant d'administration `admin` ; blocage après **10 échecs** de mot de passe.
- Scripts : exécutés uniquement depuis l'administration authentifiée, délai maximum, 3 tentatives, échec journalisé.
- HTTPS pour l'administration : option « Administration en HTTPS » de l'assistant, décochée par défaut ; cochée, certificat auto-signé créé dans `tls/` du module (`internal/wizard/tlscert.go`), remplaçable par celui du client via `tls_cert` / `tls_key`. Même port pour l'administration et l'application (mode automatique) : l'application passe aussi en https.
- Accès à l'administration : technicien (`admin`, tous les réglages) et accès client optionnel (`client`, consultation et réactivation ; 403 sur tout réglage). Sauvegarde : export sans mot de passe, import limité aux réglages fonctionnels, copie automatique avant chaque mise à jour (`sauvegardes/`).
- SmartGUARD est un outil de rappel et d'application du contrat, **pas une protection anti-piratage** : ne pas promettre le contraire dans l'interface ni la documentation.
- Ne jamais écrire de secrets (mots de passe, clés de licence, clés privées, jetons) dans les journaux ni dans le dépôt.

## Journaux

- `installation.log` : installations et mises à jour.
- `config.log` : démarrage, erreurs, actions à la date de fin.
- Chaque action d'échéance ou d'arrêt est journalisée avec son résultat.

## Conventions

- Langue de l'interface utilisateur et de la documentation : **français**. Identifiants de code et commentaires : anglais ou français, mais cohérents dans tout le dépôt.
- Petits commits, un sujet par commit, message à l'impératif.
- Tout comportement lié aux dates (échéance, J-7, J-30, changement d'heure, recul d'horloge, redémarrage) doit avoir un test avec une horloge injectable.
- Ne pas ajouter de dépendance sans nécessité : le binaire doit rester unique et léger, et éviter les faux positifs d'antivirus (pas d'empaquetage ni d'obfuscation).
- Avant de modifier le comportement décrit dans le cahier des charges, mettre à jour ce dernier et son historique de versions.

## Points encore ouverts

Voir la section 16 du cahier des charges, notamment : prix des options (section 18), vérification ARTCI avant la mise en service du portail, renouvellement et révocation des licences SmartGUARD via le portail.
