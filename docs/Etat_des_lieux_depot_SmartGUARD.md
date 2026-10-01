# ÉTAT DES LIEUX DU DÉPÔT — SmartGUARD

| Date | Dépôt analysé | Référence projet |
|---|---|---|
| 1er octobre 2026 | `C:\Users\HP\Desktop\SMARTGUARD` (branche `main`, 13 commits, version 1.6.0) | Cahier des charges v1.17, plan de développement v1.2 |

## 1. Constat principal

Le dépôt contient déjà **un module SmartGUARD fonctionnel (v1.6.0)**, développé sur la base d'un **autre cahier des charges** (v1.6, présent dans le dépôt), différent de la version de référence du projet (v1.17). Le dépôt a aussi son propre `CLAUDE.md`, qui diverge de celui du projet.

- Environ 4 300 lignes de Go ; tests présents pour scheduler, config, actions, admin, licence et proxy.
- `go test ./...` : **tous les tests passent** (vérifié sous Linux). `gofmt` : rien à signaler.
- Compilation Windows non vérifiable dans l'espace de travail de Claude (téléchargement de `golang.org/x/sys` bloqué) : à vérifier sur le poste.
- **Aucun dépôt distant configuré** dans `.git/config` : le dépôt local n'est pas relié à `github.com/henriszadi-sys/smartguard`.
- Des exécutables et une archive (`SmartGUARD-Setup-v1.6.zip`, 8,7 Mo) sont versionnés dans le dépôt.

## 2. Ce qui existe déjà (correspond aux lots 1 à 5 du plan)

| Élément | État |
|---|---|
| Configuration JSON par module, état persistant (`config.state.json`) | Fait |
| Une échéance par module (`end_date`), calcul à 00:00 heure du serveur, J-7 rouge | Fait |
| Anti-recul d'horloge, actions exécutées une seule fois, rattrapage au démarrage | Fait |
| Actions : arrêt de services, blocage d'adresses, scripts (3 tentatives, délai) | Fait |
| Proxy automatique, bandeau, page « accès suspendu », mode ligne de code | Fait |
| Administration web : connexion `admin`, blocage après 10 échecs, HTTPS possible | Fait |
| Assistant d'installation en six étapes, service Windows / systemd (`kardianos/service`), plusieurs modules par serveur | Fait |
| Licence par poste (= serveur), clés signées Ed25519, outil fournisseur `sgkeys` | Fait (licence informative) |
| Journaux `config.log` et `installation.log` | Fait |

## 3. Écarts avec le cahier des charges v1.17

| Sujet | Dépôt (v1.6) | Projet (v1.17) |
|---|---|---|
| Dates | Une seule date : la fin **est** la date d'arrêt ; arrêt par case à cocher | Date d'arrêt planifiée **optionnelle et distincte** si elle diffère de la fin |
| Nombre d'échéances | Une par module | **Plusieurs** par module (licence, contrat, abonnement), chacune avec ses rappels, messages et actions |
| Licence SmartGUARD | Informative : ne bloque jamais l'installation ni le fonctionnement | **Obligatoire** : le module ne s'installe pas sans licence valide |
| Rôles | Un seul compte `admin` | Technicien du fournisseur (réglages réservés) et accès client limité |
| Portail SaaS, enrôlement, signal, codes de renouvellement | Absent | Lots 6 et 7 |
| Espace client, détection de panne, lien « Signaler un problème » | Absent | Lot 8a |
| Systèmes minimaux | Non précisé | Windows 10 / Windows Server 2016, Linux systemd |

## 4. Conséquence sur le plan

Les lots 1 à 5 ne partent pas de zéro : ils deviennent des **lots d'adaptation** du code existant aux écarts de la section 3. Le code existant est conservé et testé ; il n'est pas réécrit.

## 5. Décisions nécessaires

1. Quel cahier des charges fait référence : v1.17 (projet) ou v1.6 (dépôt) ?
2. Date d'arrêt : une seule date avec option d'arrêt (dépôt) ou date d'arrêt optionnelle distincte (v1.17) ?
3. Licence SmartGUARD : informative (dépôt) ou bloquante à l'installation (v1.17) ?
