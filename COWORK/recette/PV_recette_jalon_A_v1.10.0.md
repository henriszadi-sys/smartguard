# PROCÈS-VERBAL DE RECETTE — Jalon A

| Livrable | Version | Cahier des charges | Date de préparation | Statut |
|---|---|---|---|---|
| Module SmartGUARD (lots 0 à 5) | **1.10.0** (GitHub `main`, commit `b3b408d`) | v1.20 | 2 octobre 2026 | **Recette sur machines réelles à faire** |

*Document préparé par Claude à partir des tests automatiques et des essais déjà réalisés. Les colonnes « Machine réelle » sont à remplir par le testeur (développeur ou Labenie). Règle : un critère n'est déclaré conforme sur machine réelle qu'après exécution du scénario et avec une preuve (capture, extrait de journal).*

## 1. Environnements

| Environnement | Utilisé à ce jour | À utiliser pour la recette |
|---|---|---|
| Linux (sans systemd), Go 1.24 | Tests automatiques, essais navigateur (Chromium), module en HTTP et HTTPS | — |
| Windows 10 (build 19045), Go 1.27 — PC de Labenie | Compilation, tests automatiques, contrôle du système, licence signée réelle | Oui (poste client Windows 10) |
| Windows Server 2016 (machine virtuelle d'évaluation) | Non | **Oui** |
| Linux avec systemd (Ubuntu 20.04+ ou Debian 11+) | Non | **Oui** |
| Serveur sans accès Internet | Non | **Oui** (fonctionnement hors ligne) |

Prérequis : exécutable v1.10.0 construit depuis `main` (`go build`), une licence SmartGUARD signée émise avec `sgkeys` (voir `EMISSION_CLES.md`), une petite application web de test (ex. Tomcat ou un serveur web) et un service système de test à arrêter.

## 2. Périmètre

**Dans le jalon A (module installé chez le client)** : critères C1 à C5, C7 à C16 et C20 de la section 15 du cahier des charges v1.20 (numérotation dans l'ordre de la section), plus la partie « module non rattaché » de C17.

**Hors jalon A** (portail, options — lots 6 à 8) : C6, C17 (rattachement), C18, C19, C21 à C33 ; dans C3, l'interdiction d'une même licence active sur deux serveurs (activation en ligne, lot 7) ; dans C5, la partie « éditeur / impayé » côté portail.

## 3. Tableau des critères

Statut à ce jour : **Conforme (tests)** = vérifié par tests automatiques sous Linux et Windows, et le cas échéant par un essai réel ; **Partiel** = une partie seulement est vérifiable sans machine réelle ; **Non testé** = à faire.

| N° | Critère (résumé) | Preuve obtenue à ce jour | Statut à ce jour | Scénario machine réelle | Résultat machine réelle | Statut final |
|---|---|---|---|---|---|---|
| C1 | Assistant en six étapes, récapitulatif avant installation | Essai navigateur complet de l'assistant (6 étapes, récapitulatif, installation) sous Linux | Partiel | S1 |  |  |
| C2 | Service créé et démarré automatiquement sous Windows ; Linux avec ou sans écran | Assistant sans écran utilisé sous Linux ; service non testé sous Windows | Non testé | S1, S2, S3 |  |  |
| C3 | Pas d'installation sans licence valide ; module hors ligne une fois activé | `TestNewInstallRequiresLicense`, `TestRequireForInstall`, essai navigateur (refus sans clé, installation avec clé, clé chargée par fichier) ; commande `install` refusée sans licence | Conforme (tests) — sauf « deux serveurs » (lot 7) | S4, S12 |  |  |
| C4 | Licences signées uniquement ; mise à jour sans licence signalée ; module installé jamais arrêté par la licence | `TestEmbeddedPublicKeyRejectsUnsignedKeys` avec une vraie licence émise (Windows), `TestUpdateWithoutLicense`, essai : licence retirée, module toujours actif | Conforme (tests) | S4, S5 |  |  |
| C5 | Arrêts et blocages uniquement par les actions du fournisseur | Revue du code : aucune action déclenchée par la licence ; `TestDisabledModuleRunsNoAction`, `TestContractEndWithoutStopOptionRunsNoAction` | Conforme (tests) — partie portail hors jalon | S5, S8 |  |  |
| C7 | Réglages du fournisseur non modifiables avec un accès client | `TestClientAccessIsReadOnlyPlusRestore` (refus 403 des réglages, licence, export, import), essai navigateur en accès client | Conforme (tests) | S10 |  |  |
| C8 | Installation et fonctionnement sur Windows 10, Server 2016, Linux systemd | Contrôle du système (`check`) : Windows 10.0 build 19045 reconnu ; Linux sans systemd refusé (`TestInstallRefusedOnUnsupportedSystem`) | Partiel | S1, S2, S3 |  |  |
| C9 | Rappel au délai configuré, jours restants, orange puis rouge à J-7 | `TestWarningLevels`, `TestMessageShowsRemainingDays`, `TestDaysLeftAcrossDSTChange`, essai navigateur du bandeau | Conforme (tests) | S6 |  |  |
| C10 | Modes automatique et ligne de code ; test du mode automatique | `TestInjectsBannerAndRewritesRedirects`, essais navigateur (proxy et ligne de code) | Conforme (tests) | S6 |  |  |
| C11 | Actions exécutées et journalisées à l'échéance | `TestActionsRunOnceAtExpiry`, `TestEachDeadlineRunsItsOwnActionsOnce`, essai : journal « ARRÊT PLANIFIÉ » (services d'essai inexistants sous Linux) | Partiel (vrais services non testés) | S8 |  |  |
| C12 | Message « expiré » ; aucune action sans l'option d'arrêt | `TestExpiryAtMidnightOfEndDate`, `TestContractEndWithoutStopOptionRunsNoAction` | Conforme (tests) | S7 |  |  |
| C13 | Avec l'option : services, adresses, scripts une seule fois et journalisés | `TestMissedActionsRunOnceAfterRestart`, `TestClockRollbackAfterExpiryDoesNotRetrigger`, `TestScriptRetriedThreeTimesThenFails`, `TestBlockedPageNamesDeadline`, essai navigateur de la page « accès suspendu » | Partiel (vrais services non testés) | S8, S9 |  |  |
| C14 | Renouvellement et réactivation depuis l'administration | `TestRestoreSingleDeadline`, `TestRestoreOfferedAfterRenewal`, `TestRestoreSkipsStillStoppedDeadlines`, essai navigateur (renouvellement puis réactivation) | Partiel (vrais services non testés) | S9 |  |  |
| C15 | Plusieurs modules sur un serveur, ports distincts | Licence du serveur partagée entre modules (`TestLegacyModuleLicenseBecomesServerLicense`) ; deux modules non installés ensemble | Non testé | S11 |  |  |
| C16 | Protections de connexion et diagnostics | `TestLoginLocksAfterTenFailures`, `TestProtectedAPIRequiresSessionAndHeader`, `TestMaskHidesKey`, HTTPS testé (certificat auto-signé), commande `check` sous Windows et Linux | Conforme (tests) | S10, S12 |  |  |
| C17 (partie) | Module non rattaché au portail : fonctionnement complet | Aucun code de portail dans le module ; tous les tests sans réseau | Conforme (tests) | S12 |  |  |
| C20 | Plusieurs échéances, chacune avec ses dates, rappels, messages et actions | `TestMultipleDeadlinesMostUrgent`, `TestBlocksOnlyStoppedDeadlines`, `TestMigrateLegacyConfig`, `TestConfigAPISavesDeadlines`, essais navigateur | Conforme (tests) | S6, S8 |  |  |

Total des tests automatiques : **70**, tous réussis sous Linux et sous Windows 10 (commit `b3b408d`).

## 4. Scénarios sur machines réelles

Pour chaque scénario : noter la date, la machine, le résultat obtenu et joindre la preuve (capture d'écran, extrait de `config.log` ou `installation.log`).

**S1 — Installation Windows Server 2016 (et Windows 10).** Lancer l'exécutable en administrateur ; dérouler les six étapes ; à l'étape 5, saisir la licence signée. *Attendu* : récapitulatif complet ; service créé au nom du module, en démarrage automatique (`services.msc`) ; page d'administration accessible ; `installation.log` sans clé en clair.

**S2 — Redémarrage du serveur.** Redémarrer le serveur Windows. *Attendu* : le service redémarre seul ; le bandeau est toujours présent.

**S3 — Linux systemd, sans écran.** `sudo ./smartguard-setup` sur un serveur sans interface ; ouvrir l'adresse affichée depuis un autre poste ; installer. *Attendu* : unité systemd active (`systemctl status <module>`), redémarrage automatique.

**S4 — Licence.** (a) Nouvelle installation sans clé, puis avec une clé `SGRD-…` : refus. (b) Avec la licence signée : installation. (c) Second module sur le même serveur : aucune clé demandée. (d) Clé chargée depuis un fichier texte. (e) Transfert : désactivation dans l'administration, activation sur un autre serveur. *Attendu* : conforme au cahier des charges section 4.

**S5 — Licence retirée après installation.** Désactiver la licence du serveur. *Attendu* : le module continue (bandeau, échéances, actions) ; l'administration indique la licence non activée.

**S6 — Rappels.** Créer trois échéances (contrat à J+20, licence à J+6, abonnement à J+40), en mode automatique puis en mode ligne de code. *Attendu* : bandeau rouge sur la licence (J-6), seconde ligne avec le contrat ; abonnement non affiché ; bouton « Tester » du mode automatique opérationnel.

**S7 — Fin sans option d'arrêt.** Échéance à la date du jour, option décochée. *Attendu* : message « expiré », aucune action dans `config.log`.

**S8 — Fin avec option d'arrêt (vrais services).** Échéance avec un service de test, une adresse bloquée et un script ; date de fin = demain ; avancer l'horloge du serveur au lendemain 00:01 (ou attendre). *Attendu* : service arrêté et désactivé, adresse bloquée avec la page « accès suspendu » qui nomme l'échéance, script exécuté, une seule fois ; ramener l'horloge en arrière : aucune nouvelle exécution.

**S9 — Renouvellement et réactivation.** Après S8, saisir une nouvelle date de fin, enregistrer, cliquer sur « Réactiver les services de cette échéance ». *Attendu* : service redémarré et en démarrage automatique ; journal de la réactivation ; aucune réactivation possible avant l'enregistrement de la nouvelle date.

**S10 — Accès et sécurité.** Activer l'accès client ; se connecter en client. *Attendu* : consultation et réactivation possibles, aucun réglage modifiable. 10 mauvais mots de passe : blocage. Option HTTPS cochée dans l'assistant : administration en https (avertissement du navigateur à accepter).

**S11 — Plusieurs modules.** Installer deux modules (deux logiciels) sur des ports différents. *Attendu* : deux services distincts, deux administrations, une seule licence du serveur.

**S12 — Hors ligne et diagnostic.** Couper l'accès Internet du serveur, redémarrer. *Attendu* : module pleinement fonctionnel ; commande `check` : système, configuration, port, journal en [OK].

**Sauvegarde (complément C16).** Exporter la configuration, la modifier, la réimporter ; mettre à jour le module par l'assistant. *Attendu* : export sans mot de passe ; copie dans `sauvegardes/` avant la mise à jour.

## 5. Anomalies et points connus

| N° | Description | Gravité | Reproduction | Suite |
|---|---|---|---|---|
| A1 | À l'étape 3 de l'assistant, l'adresse d'accès indiquée commence toujours par `http://`, même si l'option HTTPS est cochée ensuite à l'étape 5 (le récapitulatif et l'adresse finale sont corrects). | Mineure | Cocher HTTPS à l'étape 5 puis revenir à l'étape 3 | Correction à prévoir |
| P1 | Une même licence peut être activée sur deux serveurs tant que le portail n'existe pas. | — (point prévu, lot 7) | — | Activation en ligne avec le portail |
| P2 | La numérotation des critères (C1…) du plan de développement a été établie sur une version antérieure du cahier ; ce procès-verbal suit la numérotation de la v1.20. | — | — | Mettre à jour les références du plan |

Aucune anomalie bloquante connue à ce jour.

## 6. Décision proposée

**Non prononcée.** Les tests automatiques et les essais réalisés ne révèlent aucune anomalie bloquante. La décision (accepté, accepté avec réserves, refusé) sera proposée après les scénarios S1 à S12 sur Windows Server 2016, Windows 10 et Linux systemd.

| Rôle | Nom | Date | Signature |
|---|---|---|---|
| Testeur (développeur) | | | |
| Chef de projet | Labenie |  |  |
