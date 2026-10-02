# CAHIER DES CHARGES — SmartGUARD

**Module de rappel d'expiration et d'application d'échéance contractuelle**

| Version | Date | Base |
|---|---|---|
| 1.20 | 2 octobre 2026 | LISEZMOI.md et demandes complémentaires (v1.2 : renommage en SmartGUARD ; v1.3 : décisions par défaut ; v1.4 : portail fournisseur en SaaS ; v1.5 : rôle de l'éditeur, abonnement, application mobile ; v1.6 : signalement d'incidents par le client ; v1.7 : décisions sur l'espace client et risques identifiés ; v1.8 : licences et contrats ; v1.9 : détection de panne confirmée par le client ; v1.10 : validation des points 28, 29 et 31 ; v1.11 : poste serveur, administration par le fournisseur, systèmes minimaux, fonctionnalités optionnelles ; v1.12 : activation du module liée à l'abonnement, Windows Server 2016, confidentialité des noms de clients ; v1.13 : licence SmartGUARD par serveur ; v1.14 : maîtrise des blocages par le fournisseur, prix estimés ; v1.15 : licence perpétuelle, accès de base, proposition de tarification ; v1.16 : abonnement en trois formules ; v1.17 : options proportionnelles aux formules ; v1.18 : date de fin = date d'arrêt, alignement sur le code existant ; v1.19 : licences SmartGUARD signées par l'éditeur, mise à jour sans licence ; v1.20 : HTTPS en option à l'installation) |

*Document de spécification fonctionnelle*

---

## 1. Objet et contexte

SmartGUARD est un module installé sur un serveur pour suivre la date de fin d'une **licence** ou d'un **contrat** (support technique, maintenance, abonnement) lié à un logiciel. Il avertit les utilisateurs à l'approche de cette date puis peut, à la date de fin et si l'option d'arrêt est cochée, exécuter des actions définies par un technicien : arrêt de l'application ou de services, blocage d'adresses et exécution de scripts.

Le produit comprend un assistant d'installation et de configuration, un service fonctionnant en arrière-plan, une interface web d'administration, une **licence SmartGUARD par serveur** et un **portail fournisseur** proposé en SaaS par l'éditeur de SmartGUARD, qui permet au fournisseur de suivre les dates de validité des contrats et licences de tous ses clients et d'en être averti, ainsi qu'une **application mobile** optionnelle pour le fournisseur. En option, un **espace client** permet au client de signaler au fournisseur une panne ou un dysfonctionnement de son application et de suivre le traitement de sa demande.

> **Terminologie :** dans ce document, une **échéance** désigne la date de fin de validité d'une licence ou d'un contrat (support technique, maintenance, abonnement). Sauf mention contraire, ce qui est dit d'un contrat s'applique aussi à une licence. Deux notions de licence coexistent et sont toujours qualifiées : la **licence du logiciel** (ou des services) que le fournisseur a déployés chez son client, suivie par SmartGUARD comme échéance ; et la **licence SmartGUARD**, droit d'utiliser SmartGUARD sur un serveur (section 4). Employé seul, « licence » désigne la licence du logiciel du fournisseur.

## 2. Objectifs

- Configurer un module par logiciel surveillé et administrer plusieurs modules indépendants sur un serveur.
- Afficher un avertissement avant la fin d'une licence ou d'un contrat puis appliquer les actions convenues aux dates prévues.
- Permettre le renouvellement de la licence ou du contrat, la réactivation des services et le diagnostic du module.
- Exiger une licence SmartGUARD pour chaque serveur de l'application à contrôler, rattachée au compte du fournisseur.
- Informer le fournisseur des dates de fin de licence et de contrat et lui permettre de consulter les échéances de l'ensemble de ses clients depuis un portail unique.
- Améliorer la communication du support technique : permettre au client de signaler un incident au fournisseur et de suivre sa prise en charge.

## 3. Utilisateurs et rôles

**Administrateur du serveur (côté client)** — Gère la machine du client. Il peut consulter l'administration de SmartGUARD en lecture, saisir un code de renouvellement et déclencher la réactivation, mais ne modifie pas les réglages réservés au fournisseur.

**Technicien du fournisseur (administrateur SmartGUARD)** — Installe, modifie, désinstalle et diagnostique SmartGUARD sur le poste serveur du client. Il détient le mot de passe d'administration et gère les échéances et leur option d'arrêt, l'activation de la licence SmartGUARD, les ports, et les réglages réservés au fournisseur (section 10).

**Utilisateur de l'application** — Utilise l'application et voit, à l'approche de l'échéance, le message de rappel configuré.

**Fournisseur** — Entreprise qui a vendu au client le logiciel surveillé et le contrat de support (directement, sans revendeur). Il dispose d'un compte sur le portail fournisseur, reçoit les alertes d'échéance, consulte les échéances de ses clients et émet les codes de renouvellement ; avec l'option espace client, il reçoit et traite les incidents signalés par ses clients. Un compte fournisseur peut comporter plusieurs utilisateurs (gestionnaire, commercial, lecture seule).

**Contact client (option espace client)** — Personne désignée chez le client (responsable informatique, référent de l'application) qui signale les incidents au fournisseur et suit leur traitement. Elle n'a pas accès aux autres clients du fournisseur.

**Éditeur de SmartGUARD (opérateur du SaaS)** — Conçoit SmartGUARD, héberge et exploite le portail fournisseur et l'application mobile : création des comptes fournisseurs, facturation des abonnements, supervision et support.

> **Modèle commercial :** il n'y a pas de revendeur. L'éditeur met le SaaS à la disposition des fournisseurs, qui paient un abonnement. Le fournisseur et son client conservent leurs moyens de paiement habituels : SmartGUARD ne facture ni n'encaisse rien entre eux.

## 4. Licence SmartGUARD par serveur

- Un **poste** (ou serveur) est la machine, physique ou virtuelle, qui sert de serveur chez le client : le logiciel du fournisseur à contrôler y est déjà installé, et SmartGUARD y est installé à son tour.
- **Une licence SmartGUARD est obligatoire pour chaque serveur de l'application à contrôler.** Elle est acquise par le fournisseur auprès de l'éditeur et rattachée à son compte ; elle s'ajoute à l'abonnement au SaaS.
- Une licence SmartGUARD couvre tous les modules installés sur ce serveur.
- **Activation** : à l'installation, le technicien du fournisseur saisit la clé de licence SmartGUARD ou charge le fichier de licence ; activation hors ligne dès maintenant, activation en ligne avec le portail. Sans licence valide, **une nouvelle installation est refusée** (assistant et commande d'installation).
- **Mise à jour d'un module déjà installé** : possible sans licence SmartGUARD active ; l'assistant et l'administration la signalent alors comme « licence à activer ». *(validé le 2 octobre 2026)*
- **Licences signées par l'éditeur** : chaque clé de licence est signée par l'éditeur (Ed25519) et vérifiée hors ligne par le module grâce à la clé publique intégrée au programme ; les clés non signées sont refusées. La clé privée de l'éditeur est conservée par lui, hors du code et du dépôt. *(validé le 2 octobre 2026)*
- Une fois le module installé, la licence SmartGUARD **ne l'arrête jamais** : sa disparition ou sa désactivation n'interrompt ni les rappels ni les actions configurées par le fournisseur.
- La licence est liée au serveur par une empreinte matérielle ; elle ne peut pas être active sur deux serveurs en même temps. Changement de serveur : désactivation sur l'ancien, puis activation sur le nouveau.
- Une fois activé, le module **fonctionne hors ligne** et ne dépend pas du portail ; le rattachement au portail reste optionnel (section 13.2).
- **Maîtrise des arrêts et blocages par le fournisseur** : le fournisseur achète la licence SmartGUARD et en dispose ; c'est lui, et lui seul, qui décide d'arrêter ou de bloquer l'application de son client, au moyen des actions qu'il configure (sections 8 et 10). L'éditeur ne déclenche jamais lui-même un arrêt ou un blocage chez le client, même si le fournisseur cesse de payer sa licence SmartGUARD ou son abonnement ; les conséquences d'un impayé restent entre l'éditeur et le fournisseur (point 36).
- **Licence perpétuelle** : 150 000 F CFA par serveur, payée une fois. Elle comprend le module SmartGUARD complet, l'**accès de base** à la plateforme (section 13.1) et une **notification par e-mail 30 jours avant chaque échéance** (contrat de support ou licence du logiciel). Les autres options sont vendues à part (section 18).
- **Mises à jour** : les correctifs et mises à jour de sécurité sont inclus ; les nouvelles versions majeures sont payantes (prix de mise à niveau section 18).
- Le portail liste les licences SmartGUARD du fournisseur : libres, activées (serveur et client associés), à renouveler.
- L'administration du module affiche l'état de la licence SmartGUARD (active, à renouveler, expirée).

## 5. Assistant d'installation

L'assistant guide l'administrateur en six étapes :

1. **Logiciel** : nom du logiciel, nom du module (proposé automatiquement) et contact du fournisseur (nom, e-mail, téléphone). Option **« Rattacher au portail fournisseur »** : saisie de la clé d'enrôlement remise par le fournisseur et test de connexion au portail.
2. **Échéances** : pour chaque échéance suivie, type (**licence**, **contrat de support / maintenance**, **abonnement**), dates de début et de fin, option **« Arrêter l'application à la date de fin »**, délai de rappel (30 jours par défaut), activation et aperçu du message. Un module peut suivre plusieurs échéances d'un même logiciel (par exemple sa licence et son contrat de support).
3. **Affichage** : mode automatique avec bouton de test, ou intégration par ligne de code.
4. **À l'échéance** : services à arrêter, adresses à bloquer et scripts à exécuter.
5. **Sécurité** : clé de licence SmartGUARD du serveur (saisie ou fichier) si le serveur n'en a pas encore, mot de passe administrateur, option « Administration en HTTPS » (décochée par défaut), dossier d'installation et raccourci éventuel.
6. **Récapitulatif** : revue de la configuration, y compris l'état du rattachement au portail et la liste des informations transmises au fournisseur, et confirmation avant installation.

À la fin, l'assistant affiche l'adresse d'administration et, en mode ligne de code, le code à intégrer à l'application.

## 6. Installation et fonctionnement en service

- Sous Windows Server, l'installation crée un service portant le nom du module, configuré pour démarrer automatiquement avec le serveur.
- Sous Linux, l'installation peut être lancée en ligne de commande. Sans écran, l'assistant fournit une adresse accessible depuis un autre poste.
- Plusieurs modules peuvent fonctionner sur le même serveur, chacun surveillant un logiciel et utilisant son propre port.

## 7. Rappels et modes d'affichage

Le rappel commence au nombre de jours défini avant la date de fin de chaque échéance. Le texte par défaut dépend du type :

- **Contrat de support / maintenance** : « L'assistance et le support technique à votre logiciel prendra fin dans x jours, veuillez contacter le fournisseur ».
- **Licence** : « La licence de votre logiciel expire dans x jours, veuillez contacter le fournisseur ».
- **Abonnement** : « L'abonnement à votre logiciel prend fin dans x jours, veuillez contacter le fournisseur ».

Le texte reste modifiable. Il indique le nombre de jours restants, apparaît en orange puis en rouge à partir de J-7.

- **Automatique** : SmartGUARD se place devant l'application et ajoute le message à ses pages. Le port est configurable ; si SmartGUARD reprend le port courant, l'application doit être déplacée vers un autre port.
- **Ligne de code** : l'application intègre le code fourni par l'assistant pour afficher le message.

## 8. Expiration et arrêt

### Date de fin = date d'arrêt

Pour chaque échéance, il n'existe qu'une seule date : la date de fin de la licence ou du contrat. Elle est aussi la date d'arrêt. Elle est saisissable dans l'assistant et modifiable dans l'administration.

L'échéance survient à 00:00 le jour de la date de fin. À ce moment, le message de rappel de cette échéance passe à l'état « expiré ».

### Option « Arrêter l'application à la date de fin »

Case à cocher propre à chaque échéance, décochée tant que le technicien du fournisseur ne l'a pas activée.

- **Cochée** : à 00:00 le jour de la date de fin, SmartGUARD exécute une seule fois les actions configurées pour cette échéance (arrêt et désactivation des services sélectionnés, blocage des adresses avec page « accès suspendu », exécution des scripts) et consigne l'exécution ainsi que toute erreur dans son journal.
- **Décochée** : seul le message passe à « expiré » ; aucune action n'est jamais déclenchée automatiquement.
- Le recul de l'horloge du serveur ne repousse pas l'arrêt.
- Migration : dans une configuration issue d'une version antérieure, l'ancienne date d'arrêt (`stop_date`) est reprise comme date de fin et l'option est cochée.

## 9. Renouvellement et réactivation

L'administrateur peut saisir une nouvelle date de fin de licence ou de contrat dans l'interface d'administration. Après enregistrement, il peut déclencher explicitement l'action « Réactiver les services » pour remettre en service les services concernés.

Pour un module rattaché au portail, le renouvellement se fait par **code de renouvellement signé** :

- Le fournisseur génère le code depuis le portail. Le code contient l'identifiant du module, la nouvelle date de fin et une date d'émission, et il est signé avec la clé du compte fournisseur.
- L'administrateur du client saisit le code dans l'administration. Le module vérifie la signature et l'identifiant, puis enregistre la nouvelle date.
- Le code fonctionne hors ligne, ne vaut que pour un seul module et ne peut pas avancer la date de fin au-delà de celle qu'il contient.
- La réactivation des services reste une action **explicite** de l'administrateur après l'enregistrement.
- La saisie manuelle d'une nouvelle date reste possible pour un module non rattaché. Pour un module rattaché, elle est autorisée mais signalée au fournisseur.

## 10. Administration, modification et désinstallation

- Connexion à l'interface web avec l'identifiant `admin` et le mot de passe défini à l'installation.
- Consultation et modification des dates, de l'activation, des actions configurées et du journal.
- Affichage, pour chaque échéance, de la date de fin et de l'option d'arrêt, ainsi que de l'état de la licence SmartGUARD du serveur.
- **Réglages réservés au fournisseur** (technicien du fournisseur, dans l'administration du module) : page « accès suspendu », scripts lancés à l'échéance, comportement au redémarrage du serveur autour de l'échéance, sauvegardes et restauration de la configuration. Ces réglages ne sont pas modifiables par le client, ni à distance depuis le portail.
- Affichage de l'état du rattachement au portail (rattaché ou non, dernier envoi réussi, informations transmises), avec possibilité de rattacher, de détacher ou d'exporter un fichier d'état pour le fournisseur.
- Saisie d'un code de renouvellement signé.
- Depuis l'écran d'accueil : modifier les paramètres, ouvrir l'administration ou désinstaller le module.
- La modification reprend les valeurs existantes ; renommer un module renomme également le service correspondant.
- La désinstallation supprime le service, la règle de pare-feu et les fichiers du module. Pour un module rattaché, elle envoie au portail un dernier message « module désinstallé » si la connexion le permet.

## 11. Sécurité

- L'assistant d'installation est accessible uniquement avec le lien secret affiché par son programme.
- Le mot de passe administrateur est stocké sous forme hachée, sans possibilité de récupération en clair.
- L'accès est bloqué après 10 échecs de mot de passe.
- Le recul de l'horloge du serveur ne doit pas repousser l'échéance.
- Un administrateur du serveur peut arrêter le module ; SmartGUARD est un outil de rappel et d'application du contrat, pas une protection anti-piratage.
- Liaison licence / serveur : identifiant de la machine enregistré à l'activation, installation copiée sur un autre serveur détectée. Sans portail, le module vérifie seulement la signature de l'éditeur : empêcher qu'une même licence soit active sur deux serveurs demande l'activation en ligne (portail). Comme le reste du produit, ce n'est pas une protection anti-piratage.
- Échanges module ↔ portail uniquement en HTTPS sortant depuis le serveur du client ; aucun port entrant n'est ouvert pour le portail.
- Chaque module s'authentifie auprès du portail avec un jeton propre, obtenu à l'enrôlement et révocable depuis le portail.
- Les codes de renouvellement sont signés par une clé propre à chaque compte fournisseur ; le module conserve la clé publique reçue à l'enrôlement.

## 12. Journaux et diagnostic

- `installation.log` : détail des installations et mises à jour.
- `config.log` : démarrage, erreurs et actions à l'expiration ou à l'arrêt à la date de fin, ainsi que les envois au portail et la saisie des codes de renouvellement.
- Diagnostic depuis l'assistant : configuration, port, application, état du service, journal d'événements Windows et connexion au portail.
- Commande manuelle de diagnostic Windows : exécuter le programme installé avec le paramètre de configuration puis la commande `check`, en invite administrateur.
- Commandes technicien optionnelles : consulter l'état, changer le mot de passe et lancer un diagnostic complet.

## 13. Portail fournisseur (SaaS)

Le portail fournisseur est un service en ligne hébergé et exploité par l'éditeur de SmartGUARD, auquel le fournisseur s'abonne. Il lui permet de suivre les dates de validité des contrats et licences de ses clients, ainsi que les autres informations utiles à leur suivi. Il est **multi-fournisseurs** : chaque fournisseur ne voit que ses propres clients et modules. Le rattachement d'un module au portail est **optionnel**.

### 13.1 Comptes et abonnements

- Création d'un compte fournisseur par l'éditeur ou par inscription.
- **Accès de base** (inclus avec toute licence SmartGUARD, sans abonnement) : compte fournisseur, saisie des clients et de leurs échéances (dates de début et de fin des contrats et licences), gestion des licences SmartGUARD, et **une seule notification par e-mail 30 jours avant chaque échéance**.
- **Abonnement** (par compte fournisseur, en trois formules selon le nombre de serveurs rattachés au portail : Essentiel, Pro, Entreprise ; section 18.2) : débloque le portail complet décrit dans cette section : toutes les alertes (section 13.3), tableau de bord, calendrier, fiches clients, exports, codes de renouvellement signés, signal des modules rattachés.
- En cas d'abonnement expiré ou impayé, le compte revient à l'accès de base après un délai de grâce (la notification à J-30 continue) ; **les modules installés chez les clients continuent de fonctionner normalement**.
- Plusieurs utilisateurs par compte, avec trois rôles : gestionnaire (tous les droits, y compris codes de renouvellement), commercial (consultation et alertes), lecture seule.
- Connexion par e-mail et mot de passe ; double authentification proposée.
- Génération de clés d'enrôlement, chacune liée à un client du fournisseur.

### 13.2 Rattachement des modules

- **Enrôlement** : à l'installation (étape 1 de l'assistant) ou plus tard depuis l'administration, le module présente la clé d'enrôlement et reçoit un jeton d'accès et la clé publique du fournisseur.
- **Signal régulier** : toutes les 6 heures, et à chaque événement important, le module envoie au portail :
  - identifiants du client, du logiciel et du module ;
  - échéances suivies (type licence, contrat ou abonnement, date de fin et option d'arrêt) ;
  - état de chaque échéance : active, en rappel, expirée ; état du module : actif, arrêté, suspendu ;
  - dernières actions exécutées et leur résultat ;
  - état de la licence SmartGUARD du serveur ;
  - version de SmartGUARD.
- **Aucune donnée métier** de l'application surveillée n'est transmise.
- En cas d'échec, le module conserve les messages et les renvoie à la connexion suivante.
- **Clients sans Internet** : le fournisseur saisit le contrat à la main dans le portail, ou importe le fichier d'état exporté depuis l'administration du module.

### 13.3 Alertes au fournisseur

Les alertes sont calculées et envoyées **par le portail**, à partir des dates connues. Le fournisseur reste donc averti même si le serveur du client est éteint ou hors ligne.

- **Accès de base (licence seule)** : une notification par e-mail à J-30 pour chaque échéance.
- **Avec abonnement** : l'ensemble des alertes ci-dessous.

- **Avant l'échéance** : à J-90, J-60, J-30, J-7 et le jour J (délais modifiables par le fournisseur).
- **Sur événement** :
  - licence ou contrat expiré et actions exécutées ou en échec ;
  - arrêt exécuté à la date de fin ;
  - renouvellement enregistré ou nouvelle date saisie à la main sur le module ;
  - recul d'horloge détecté ;
  - module silencieux depuis plus de 48 heures (délai modifiable) ;
  - module désinstallé ou détaché.
- **Récapitulatif hebdomadaire** : échéances des 90 prochains jours.
- **Canaux** : e-mail en V1, notifications de l'application mobile pour les fournisseurs qui l'ont souscrite (section 13.8) ; SMS et WhatsApp prévus dans une version ultérieure.
- Chaque utilisateur choisit les alertes qu'il reçoit.

### 13.4 Consultation des échéances

- **Tableau de bord** : toutes les licences et tous les contrats triés par date de fin, avec code couleur (vert, orange, rouge à partir de J-7, gris pour expiré) et indicateurs (licences et contrats à renouveler dans 30, 60 et 90 jours, modules silencieux).
- **Filtres** : client, logiciel, type d'échéance (licence, contrat, abonnement), statut, période ; **vue calendrier** des échéances.
- **Fiche client** : modules et dates, état de la licence SmartGUARD de chaque serveur, dernier signal reçu, historique (renouvellements, actions, alertes envoyées).
- **Export** CSV et Excel de la liste des échéances.

### 13.5 Renouvellement depuis le portail

- Le gestionnaire génère un code de renouvellement signé pour un module (voir section 9) et l'envoie au client par e-mail depuis le portail ou le lui transmet lui-même.
- Le portail conserve l'historique des codes émis et indique quand le module a confirmé la nouvelle date.
- L'envoi automatique de la nouvelle date au module, sans saisie par le client, est une option désactivée par défaut (voir section 16).

- Le portail ne gère ni devis, ni facture, ni paiement entre le fournisseur et son client.

### 13.6 Transparence envers le client

- L'assistant et l'administration du module indiquent clairement que le module est rattaché au portail du fournisseur et listent les informations transmises.
- L'administrateur du client peut détacher le module ; le fournisseur en est alors averti.

### 13.7 Exploitation du SaaS par l'éditeur

- Hébergement avec sauvegardes quotidiennes et journal d'audit des actions des utilisateurs du portail.
- Données de chaque fournisseur isolées des autres.
- **Confidentialité des clients du fournisseur** : le fournisseur voit l'identifiant et le nom de ses clients ; l'administration de l'éditeur n'affiche que les **identifiants** des clients, jamais leurs noms. Les noms restent stockés normalement pour permettre l'envoi des alertes (les e-mails au fournisseur contiennent le nom du client). Tout accès technique de l'éditeur aux données d'un fournisseur (support, maintenance) est journalisé et consultable par le fournisseur.
- Supervision par l'éditeur : modules enrôlés, envois d'alertes, erreurs.
- Facturation des abonnements des fournisseurs par l'éditeur.

### 13.8 Application mobile du fournisseur (option)

Application proposée en option de l'abonnement, pour les utilisateurs d'un compte fournisseur.

- **Notifications** : mêmes alertes que le portail (J-90 à J, contrat expiré, actions exécutées ou en échec, module silencieux, renouvellement, détachement), avec choix par utilisateur.
- **Consultation** : liste des contrats et licences triée par échéance avec le code couleur, fiche client, état de chaque module (dernier signal, état du contrat, état de la licence SmartGUARD).
- **Actions** : marquer une alerte comme traitée, appeler ou écrire au contact client ; la génération d'un code de renouvellement reste réservée au gestionnaire et demande une confirmation.
- **Connexion** : mêmes identifiants et rôles que le portail, double authentification, déconnexion à distance d'un appareil perdu.
- **Hors connexion** : affichage des dernières données reçues, avec leur date.
- L'application s'appuie sur la même interface de programmation (API) que le portail ; aucune donnée n'est conservée sur le téléphone en dehors d'un cache de consultation chiffré.

### 13.9 Espace client et signalement d'incidents (option)

Option souscrite par le fournisseur dans son abonnement, **gratuite pour ses clients** côté SmartGUARD, **disponible dès la V1**. Le fournisseur reste libre de facturer ce service à son client par son propre contrat ; SmartGUARD n'intervient pas dans cette facturation. Elle sert de canal de communication du support technique entre le client et le fournisseur. Ce n'est pas un outil complet de gestion de support : V1 limitée au **signalement simple** (déclaration, échange de messages, statut, pièces jointes, transfert par e-mail).

**Règle de conception : le signalement ne dépend jamais du serveur du client.** Les signalements sont enregistrés dans le SaaS ; le module installé chez le client ne sert qu'à proposer un raccourci et à joindre son état technique.

**Points d'entrée pour le client**

- **Espace client web** sur le SaaS, accessible depuis n'importe quel navigateur ou téléphone : c'est le point d'entrée principal, qui reste disponible quand le serveur du client est hors service.
- **Bouton « Signaler un incident »** dans l'administration du module SmartGUARD installé chez le client.
- **Lien « Signaler un problème »** ajouté dans les pages de l'application en mode automatique (même mécanisme que le bandeau de rappel), activable par le fournisseur.
- L'application mobile pour le client n'est pas prévue en V1 ; l'espace client web est adapté à l'affichage sur téléphone. Une application client pourra être ajoutée si la demande se confirme.
- L'espace client affiche en permanence le **numéro d'urgence** du fournisseur, pour les pannes qui demandent un appel immédiat.
- V1 : un accès client par fournisseur (un client qui a plusieurs fournisseurs a un accès chez chacun).

**Contenu d'un signalement**

- Gravité : **bloquant** (application inutilisable), **dégradé** (fonction en panne ou lenteur), **question** (demande d'assistance).
- Titre, description, date de début du problème, nombre d'utilisateurs touchés, pièces jointes (captures d'écran, fichiers journaux) dans une limite de taille. Au moment de l'envoi, un avertissement rappelle de ne pas joindre de données personnelles inutiles.
- Cause présumée si le client la connaît : coupure d'électricité, réseau ou Internet, matériel, logiciel, autre.
- Coordonnées de la personne à rappeler.
- **Contexte technique ajouté automatiquement** quand le module est rattaché : version, état du service, dernier signal, état du contrat, dernières actions exécutées par SmartGUARD.

**Diagnostic immédiat**

- Si l'application est indisponible parce que SmartGUARD a appliqué les actions d'arrêt d'une échéance (fin de licence ou de contrat), le client en est informé dès la saisie (avec les coordonnées du fournisseur), et le signalement est marqué comme tel pour le fournisseur.
- **Licence ou contrat expiré** : le signalement est accepté. Le client voit le message : « Support expiré, merci de le renouveler. Une équipe interviendra dans les plus brefs délais. » Le signalement est marqué **« hors contrat »** pour le fournisseur. Ce texte est celui par défaut ; le fournisseur peut le modifier.

**Le client décide toujours d'alerter le fournisseur**

- Un incident n'alerte le fournisseur qu'après une action du client : un signalement rédigé par lui, ou la confirmation d'une panne détectée automatiquement.
- Le contexte technique joint automatiquement (voir plus haut) complète le signalement du client mais ne le remplace pas.

**Détection automatique de panne, confirmée par le client (option)**

- **Activation** : désactivée par défaut ; le fournisseur l'active pour un module. Le client en est informé dans son espace et dans l'administration du module, et peut la désactiver.
- **Surveillance** : le module contrôle à intervalle régulier une adresse de l'application choisie par le fournisseur, en mode automatique comme en mode ligne de code. Une panne est retenue si l'application ne répond plus pendant une durée réglable (5 minutes par défaut).
- **Exclusions** : un arrêt provoqué par SmartGUARD (arrêt à la date de fin) n'est jamais une panne ; le client ou le fournisseur peut déclarer des plages de maintenance pendant lesquelles rien n'est détecté.
- **Prévenir le client** : la panne détectée est envoyée au service en ligne, qui prévient les contacts du client par e-mail avec un lien vers l'espace client (et par l'application client lorsqu'elle existera). Le bandeau dans l'application n'est pas utilisé, puisqu'elle est en panne.
- **Confirmation** : depuis l'espace client, le contact confirme la panne, peut la compléter (gravité, description, cause présumée, pièces jointes) puis l'envoie au fournisseur ; il peut aussi l'ignorer (fausse alerte, maintenance).
- **Avant confirmation** : le fournisseur voit la panne dans son portail avec le statut **« détectée, non confirmée »**, sans e-mail ni notification ; il peut contacter le client s'il le souhaite.
- **Après confirmation** : la panne devient un signalement normal et suit le traitement ci-dessous, avec les alertes au fournisseur.
- **Retour à la normale** : si l'application répond de nouveau avant la confirmation, la panne détectée est close automatiquement et le client en est informé ; après confirmation, le signalement reste ouvert jusqu'à sa résolution par le fournisseur.
- **Limite** : si le serveur du client est éteint ou privé d'Internet, le module ne peut ni détecter ni prévenir. Ce cas est couvert par l'alerte « module silencieux » du portail (section 13.3), qui reste envoyée au fournisseur sans confirmation du client, et par le signalement manuel depuis l'espace client.

**Traitement par le fournisseur**

- Réception sur le portail, par e-mail et par notification de l'application mobile, selon la gravité et les préférences de chaque utilisateur. Un signalement **bloquant** déclenche une notification prioritaire.
- Relance du fournisseur pour tout signalement resté sans prise en charge au-delà d'un délai qu'il choisit.
- Statuts : **nouveau → pris en charge → en cours → résolu → clos** ; le client peut rouvrir un incident résolu dans un délai donné.
- Messagerie entre le fournisseur et le client sur chaque incident ; le client est averti par e-mail à chaque changement de statut ou réponse.
- Attribution d'un incident à un utilisateur du compte fournisseur.
- Indicateurs internes au fournisseur : incidents ouverts par client et par gravité, délai de prise en charge, délai de résolution. **Aucun délai d'intervention n'est affiché au client par défaut** ; des engagements de délai (SLA) par contrat pourront être ajoutés dans une version ultérieure, à l'initiative du fournisseur.
- Historique des incidents dans la fiche client.
- Transfert de chaque nouvel incident par e-mail ou par appel web (webhook) vers l'outil de support que le fournisseur utilise déjà.

**Comptes et sécurité**

- Le fournisseur invite les contacts de chaque client et peut les désactiver ; chaque contact a son identifiant et son mot de passe, double authentification proposée.
- Un client ne voit que ses propres incidents et l'état de ses propres contrats.
- Limitation du nombre de signalements par heure pour éviter les abus ; pièces jointes analysées et conservées pour une durée limitée.

## 14. Contraintes techniques connues

- Systèmes minimaux : **Windows 10** et **Windows Server 2016**, ainsi que Linux avec systemd.
- Interface de configuration et d'administration accessible dans un navigateur.
- Un port distinct par module installé sur un serveur.
- Deux modes d'intégration à l'application : proxy automatique ou ligne de code.
- L'antivirus peut mettre le programme en quarantaine ; l'assistant doit signaler le diagnostic et la procédure documentée.
- Le module doit fonctionner normalement sans portail (installation non rattachée ou portail injoignable) : rappels, échéances et actions ne dépendent jamais du portail.
- Le portail est un service distinct, hébergé par l'éditeur, avec sa propre base de données et une interface de programmation (API) utilisée par l'interface web et par l'application mobile.

## 15. Critères d'acceptation

- L'assistant présente les six étapes et récapitule les paramètres avant installation.
- Le service est créé et démarre automatiquement sur Windows ; le fonctionnement Linux est accessible selon le mode écran ou sans écran.
- Le module ne s'installe qu'avec une licence SmartGUARD valide ; une licence ne peut pas être active sur deux serveurs en même temps ; une fois activé, le module fonctionne hors ligne.
- Seules les licences signées par l'éditeur sont acceptées ; la mise à jour d'un module déjà installé reste possible sans licence et affiche « licence à activer » ; un module installé continue de fonctionner si la licence est retirée.
- Les arrêts et blocages chez le client ne résultent que des actions configurées par le fournisseur ; l'éditeur ne peut en déclencher aucun, y compris en cas d'impayé du fournisseur.
- Les noms des clients ne sont jamais affichés dans l'administration de l'éditeur ; tout accès technique de l'éditeur aux données d'un fournisseur est journalisé.
- Les réglages réservés au fournisseur ne sont pas modifiables avec un accès client.
- SmartGUARD s'installe et fonctionne sur Windows 10, Windows Server 2016 et versions ultérieures, et sur Linux avec systemd.
- Le rappel apparaît au délai configuré, indique les jours restants et respecte les couleurs définies.
- Les modes automatique et ligne de code fonctionnent ; le mode automatique propose un test.
- À l'échéance, les actions configurées sont exécutées et journalisées.
- À la fin d'une licence ou d'un contrat, le message passe à « expiré » ; sans l'option d'arrêt, aucune action n'est exécutée.
- À la date de fin, si l'option d'arrêt est cochée, les actions configurées (services, adresses bloquées, scripts) sont exécutées une seule fois et journalisées.
- Le renouvellement et la réactivation des services sont possibles depuis l'administration.
- Plusieurs modules coexistent sur un serveur avec des ports distincts.
- Les protections de connexion et les diagnostics prévus sont disponibles.
- Un module peut être rattaché au portail avec une clé d'enrôlement, puis détaché ; un module non rattaché fonctionne entièrement.
- Le portail affiche, pour un fournisseur, toutes les échéances de ses clients, avec filtres, vue calendrier et export ; il n'affiche jamais celles d'un autre fournisseur.
- Avec la licence seule, le fournisseur reçoit une notification par e-mail à J-30 pour chaque échéance saisie ; avec l'abonnement, il reçoit par e-mail les alertes J-90, J-60, J-30, J-7 et J pour chaque licence et chaque contrat, même si le module est hors ligne.
- Un module peut suivre plusieurs échéances (licence, contrat, abonnement) avec leurs propres dates, rappels, messages et actions.
- Le fournisseur est averti lorsqu'un module ne donne plus de signal au-delà du délai configuré.
- Un code de renouvellement signé est accepté par le module visé, refusé par tout autre module et refusé s'il a été modifié.
- Un contrat d'un client sans Internet peut être suivi par saisie manuelle ou import du fichier d'état.
- Un abonnement fournisseur expiré n'interrompt jamais le fonctionnement des modules installés chez ses clients.
- Avec l'option mobile, le fournisseur reçoit sur son téléphone les mêmes alertes que par e-mail et consulte l'état de ses contrats et licences.
- Avec l'option espace client, un contact client peut signaler un incident depuis l'espace client web, depuis l'administration du module et, si activé, depuis l'application ; le fournisseur le reçoit et peut y répondre.
- Un incident signalé alors que l'application est arrêtée par SmartGUARD est identifié comme tel, pour le client comme pour le fournisseur.
- Un client ne peut jamais voir les incidents ou contrats d'un autre client.
- Le client peut signaler un incident depuis l'espace client web même lorsque son serveur est éteint ou sans Internet.
- Le fournisseur n'est alerté d'un incident qu'après une action du client (signalement ou confirmation d'une panne détectée).
- Avec la détection activée, une application qui ne répond plus au-delà du seuil fait prévenir le client ; la panne apparaît au fournisseur comme « détectée, non confirmée » sans alerte, puis l'alerte part à la confirmation du client.
- Une panne détectée est close automatiquement si l'application répond de nouveau avant confirmation ; un arrêt provoqué par SmartGUARD ou survenu pendant une plage de maintenance n'est jamais détecté comme panne.
- Avec une licence ou un contrat expiré, le signalement est accepté, le client voit le message « support expiré » et le fournisseur voit la mention « hors contrat ».

## 16. Points à préciser avant réalisation — décisions proposées

Chaque point ci-dessous porte une **décision par défaut proposée**, à valider par le commanditaire (statut : *à valider*, sauf mention *validé*).

| # | Point | Décision proposée |
|---|---|---|
| 1 | Page « accès suspendu » | **Administrée par le fournisseur.** En mode automatique (proxy), SmartGUARD répond lui-même par la page « accès suspendu » (HTTP 503) avec le contact du fournisseur ; en mode ligne de code, l'application affiche la page fournie ; blocage au niveau pare-feu en option, désactivé par défaut. *(validé le 1er octobre 2026)* |
| 2 | Scripts lancés à l'échéance | **Administrés par le fournisseur**, uniquement depuis l'administration authentifiée du module (jamais à distance depuis le portail) ; exécutés avec les droits du service, délai maximum (5 min par défaut), 3 tentatives puis erreur journalisée. *(validé le 1er octobre 2026)* |
| 3 | Redémarrage du serveur autour de l'échéance | **Administré par le fournisseur.** Par défaut, au démarrage, le service recalcule l'état et exécute une seule fois les actions manquées, puis les journalise. *(validé le 1er octobre 2026)* |
| 4 | Sauvegardes, migrations, restauration | **Administrées par le fournisseur.** Configuration dans un fichier unique exporté/importé depuis l'administration ; sauvegarde automatique avant chaque mise à jour ; restauration par import. *(validé le 1er octobre 2026)* |
| 5 | HTTPS pour l'administration | **Option à l'installation**, décochée par défaut : cochée, l'assistant crée un certificat auto-signé pour le serveur, remplaçable par le certificat du client. Le module servant sur un même port l'administration et, en mode automatique, l'application, celle-ci passe aussi en https. HTTP local uniquement autorisé pour l'assistant. *(validé le 1er octobre 2026 ; option décochée par défaut validée le 2 octobre 2026)* |
| 6 | Versions et ressources minimales | **Windows 10 et Windows Server 2016 minimum** ; Linux avec systemd (Ubuntu 20.04+, Debian 11+, RHEL 8+), navigateurs récents, 1 Go de RAM et 200 Mo disque par module. *(validé le 1er octobre 2026)* |
| 7 | Fuseau horaire et changements d'heure | Échéances calculées à 00:00 selon **l'heure du serveur du client**, stockées avec le fuseau ; Côte d'Ivoire sans changement d'heure. *(validé le 1er octobre 2026)* |
| 8 | Définition d'un « poste » | Le poste est **la machine qui sert de serveur** chez le client, où le logiciel du fournisseur est déjà installé et où SmartGUARD est installé. Le mot « licence » désigne la licence du logiciel ou des services du fournisseur, pas une licence de SmartGUARD. *(validé le 1er octobre 2026)* |
| 9 | Date d'arrêt | **Date de fin = date d'arrêt** pour chaque échéance ; l'arrêt est une option à cocher (décochée par défaut). Remplace la date d'arrêt planifiée distincte (décision du 1er octobre 2026, alignée sur le code existant v1.6). *(validé le 1er octobre 2026)* |
| 10 | Hébergement du portail fournisseur | Portail proposé en **SaaS**, hébergé et exploité par l'éditeur de SmartGUARD, multi-fournisseurs, sans revendeur. *(validé le 1er octobre 2026)* |
| 11 | Rattachement au portail | **Optionnel** ; la saisie manuelle et l'import du fichier d'état servent de solution de secours pour les clients sans Internet. *(validé le 1er octobre 2026)* |
| 12 | Transparence envers le client | L'assistant et l'administration affichent le rattachement et la liste des données transmises ; le client peut détacher le module. *(validé le 1er octobre 2026)* |
| 13 | Canaux d'alerte | E-mail en V1 ; notifications de l'application mobile (option) ; SMS et WhatsApp dans une version ultérieure. *(validé le 1er octobre 2026)* |
| 14 | Renouvellement à distance | Code de renouvellement signé saisi par le client (par défaut) ; envoi automatique de la nouvelle date par le portail en option, désactivée par défaut. *(validé le 1er octobre 2026)* |
| 15 | Fréquence du signal et délai « module silencieux » | Signal toutes les 6 heures ; alerte après 48 heures sans signal ; les deux sont modifiables par le fournisseur. *(validé le 1er octobre 2026)* |
| 16 | Protection des données et hébergement | Démarche retenue *(validé le 1er octobre 2026)* : vérifier les obligations applicables en Côte d'Ivoire (loi n° 2013-450, formalités auprès de l'ARTCI) avant mise en service et choisir la localisation de l'hébergement en conséquence. Vérification *(à faire)*. |
| 17 | Modèle et prix | Licence SmartGUARD perpétuelle **150 000 F CFA par serveur** (accès de base + notification J-30 inclus) *(validé le 1er octobre 2026)* ; abonnement par compte fournisseur en trois formules : Essentiel (≤ 10 serveurs) 20 000 / mois ou 200 000 / an, Pro (11 à 50) 60 000 / mois ou 500 000 / an, Entreprise (> 50) 100 000 / mois ou 1 000 000 / an *(validé le 1er octobre 2026)* ; prix des options, utilisateurs inclus et règles de changement de formule : section 18 *(proposition, à valider)*. |
| 18 | Paiements entre fournisseur et client | Hors du périmètre de SmartGUARD : aucun devis, facture ni encaissement entre le fournisseur et son client. *(validé le 1er octobre 2026)* |
| 19 | Application mobile : plateformes et technologie | Android et iOS avec une seule base de code en **Flutter** ; notifications via Firebase Cloud Messaging et le service de notification d'Apple ; Android en priorité si les ressources sont limitées. *(validé le 1er octobre 2026)* |
| 20 | Espace client : tarification | Option de l'abonnement du fournisseur, gratuite pour ses clients ; le fournisseur peut la refacturer à son client par son propre contrat, hors SmartGUARD. *(validé le 1er octobre 2026)* |
| 21 | Espace client : périmètre de la V1 | Signalement simple : déclaration, statuts, messagerie, pièces jointes, transfert par e-mail ou webhook ; pas d'outil complet de support, ni de SLA, ni de base de connaissances. *(validé le 1er octobre 2026)* |
| 22 | Détection automatique de panne | Maintenue, mais l'alerte au fournisseur n'est envoyée qu'après confirmation du client ; avant confirmation, panne visible dans le portail sans alerte. *(validé le 1er octobre 2026)* |
| 23 | Conservation des pièces jointes et des échanges | Conservés **six mois après l'échéance** de la licence ou du contrat concerné, puis supprimés ; export possible avant suppression. *(validé le 1er octobre 2026)* ; à rapprocher du point 16 (protection des données). |
| 24 | Espace client : canal | Espace client web hébergé dans le SaaS en V1, indépendant du serveur du client ; application mobile client plus tard si la demande se confirme. *(validé le 1er octobre 2026)* |
| 25 | Signalement avec licence ou contrat expiré | Accepté, avec le message « Support expiré, merci de le renouveler. Une équipe interviendra dans les plus brefs délais. » côté client (texte par défaut, modifiable par le fournisseur) et la mention « hors contrat » côté fournisseur. *(validé le 1er octobre 2026)* |
| 26 | Calendrier de l'espace client | Intégré dès la V1. *(validé le 1er octobre 2026)* |
| 27 | Licences et contrats | SmartGUARD suit les licences comme les contrats (support, maintenance, abonnement). *(validé le 1er octobre 2026)* |
| 28 | Plusieurs échéances par module | Un module suit une ou plusieurs échéances d'un même logiciel, chacune avec sa date, son rappel, son message et ses actions. *(validé le 1er octobre 2026)* |
| 29 | Lien « Signaler un problème » dans les pages de l'application | Activable par le fournisseur en mode automatique, désactivé par défaut. *(validé le 1er octobre 2026)* |
| 30 | Détection : activation | Désactivée par défaut ; activée par le fournisseur pour un module ; le client est informé et peut la désactiver. *(validé le 1er octobre 2026)* |
| 31 | Détection : réglages | Contrôle d'une adresse de l'application dans les deux modes ; seuil de 5 minutes par défaut, **modifiable** ; plages de maintenance ; clôture automatique au retour avant confirmation. *(validé le 1er octobre 2026)* |
| 32 | Alerte « module silencieux » | Inchangée : envoyée au fournisseur après le délai configuré, sans confirmation du client (suivi du module, pas un incident). *(validé le 1er octobre 2026)* |
| 33 | Licence SmartGUARD | **Une licence SmartGUARD par serveur** de l'application à contrôler, rattachée au compte du fournisseur, couvrant tous les modules du serveur ; activation en ligne ou par fichier ; liée au serveur ; fonctionnement hors ligne ensuite ; perpétuelle, 150 000 F CFA (voir points 17 et 36). *(validé le 1er octobre 2026)* |
| 34 | Compatibilité Windows | Minimum relevé à **Windows Server 2016** (et Windows 10), compatible avec les versions récentes de Go. *(validé le 1er octobre 2026)* |
| 35 | Confidentialité des noms de clients | Le fournisseur voit identifiant et nom ; l'administration de l'éditeur ne voit que les identifiants ; noms stockés normalement (pas de chiffrement par le fournisseur), accès techniques journalisés ; e-mails d'alerte avec le nom du client. *(validé le 1er octobre 2026)* |
| 36 | Durée de la licence SmartGUARD | **Perpétuelle**, sans expiration ; correctifs et mises à jour de sécurité inclus, versions majeures payantes. *(validé le 1er octobre 2026)* |
| 37 | Accès de base inclus dans la licence | Compte fournisseur, saisie des clients et échéances, gestion des licences SmartGUARD, une notification e-mail à J-30 par échéance, sans abonnement. *(validé le 1er octobre 2026)* |
| 39 | Mise à jour sans licence | La mise à jour d'un module déjà installé (installé avant la licence obligatoire) n'exige pas de licence SmartGUARD ; elle est acceptée et signalée « licence à activer ». Seule une nouvelle installation exige une licence valide. *(validé le 2 octobre 2026)* |
| 40 | Signature des licences | Licences SmartGUARD signées par l'éditeur (Ed25519), vérifiées hors ligne par le module ; paire de clés de l'éditeur créée le 2 octobre 2026, clé privée conservée par l'éditeur hors du dépôt. *(validé le 2 octobre 2026)* |
| 38 | Cahier des charges de référence | La présente version fait référence ; le cahier v1.6 présent dans le dépôt de code est remplacé. Le code existant (module v1.6.0) est conservé et adapté. *(validé le 1er octobre 2026)* |

## 17. Risques identifiés et réponses

Risques relevés lors de l'analyse des idées avant leur intégration, avec la réponse retenue ou proposée.

| Gravité | Risque | Réponse |
|---|---|---|
| Bloquant | Un canal de signalement passant par le serveur du client tombe en panne en même temps que lui (panne, coupure de courant ou d'Internet). | Signalements enregistrés dans le SaaS ; espace client web accessible depuis tout téléphone ou ordinateur (section 13.9). |
| Important | Le fournisseur hésite à confier la liste de ses clients et leurs échéances à un service tiers. | Données minimales, aucune donnée métier, isolation stricte entre fournisseurs, export et suppression des données à la demande, engagement contractuel de non-exploitation. |
| Important | Dépendance au SaaS pour les renouvellements signés. | Module entièrement autonome sans portail ; saisie manuelle d'une date toujours possible ; export de la clé de signature par le fournisseur *(à valider)*. |
| Important | Fausses alertes « module silencieux » dues aux coupures de courant et d'Internet, qui finissent par être ignorées. | Délai d'alerte modifiable (48 h par défaut) ; possibilité de marquer un module « en pause » *(à valider)*. |
| Important | Gros clients (banques, administrations) bloquant les connexions sortantes : portail aveugle sur les contrats les plus importants. | Saisie manuelle et import du fichier d'état exporté depuis le module. |
| Important | Dérive de l'espace client vers un outil complet de support, en concurrence avec des outils installés et avec WhatsApp et le téléphone. | V1 limitée au signalement simple ; transfert par e-mail ou webhook vers l'outil du fournisseur. |
| Important | Un client au contrat expiré obtient du support non payé. | Mention « hors contrat » côté fournisseur et invitation à renouveler côté client ; le fournisseur décide de la suite. |
| Important | Le message par défaut « Une équipe interviendra dans les plus brefs délais » engage le fournisseur à intervenir même sans contrat. | Texte modifiable par le fournisseur (point 25, validé). |
| Important | Deux échéances d'un même logiciel (licence et contrat) aux actions différentes peuvent se contredire, par exemple un contrat renouvelé mais une licence expirée. | Chaque échéance a ses propres actions et sa propre réactivation ; l'administration et le portail affichent clairement laquelle a déclenché un arrêt. |
| Important | Une trace horodatée des signalements peut être opposée au fournisseur s'il a des délais contractuels. | Aucun délai affiché au client par défaut ; affichage d'engagements de délai uniquement à l'initiative du fournisseur. |
| Important | Pièces jointes contenant des données personnelles (loi n° 2013-450, ARTCI). | Avertissement à l'envoi, taille et durée de conservation limitées (point 23), à rapprocher du point 16. |
| Important | Le blocage à l'échéance peut être vécu par le client comme une prise d'otage et engager la responsabilité du fournisseur en cas d'arrêt à tort. | Actions d'échéance configurables et journalisées ; mode « rappel seul » possible en ne configurant aucune action ; clause contractuelle encadrant la suspension recommandée au fournisseur. |
| Mineur | Une panne bloquante demande un appel immédiat, pas un message écrit. | Numéro d'urgence affiché dans l'espace client ; notification prioritaire pour un signalement bloquant. |
| Mineur | Un fournisseur qui tarde à répondre dégrade l'image du canal et de SmartGUARD. | Relance du fournisseur au-delà d'un délai qu'il choisit. |
| Bloquant | La détection ne fonctionne pas si le serveur du client est éteint ou privé d'Internet : le module ne peut ni détecter ni prévenir. | Limite documentée ; couverte par l'alerte « module silencieux » du portail et par le signalement manuel depuis l'espace client. |
| Important | Une panne détectée mais non confirmée (nuit, week-end) n'alerte pas le fournisseur. | Panne visible dans le portail comme « détectée, non confirmée », sans alerte ; le fournisseur peut contacter le client. |
| Important | Fausses détections : maintenance, redémarrage, lenteur passagère. | Seuil réglable, plages de maintenance, exclusion des arrêts provoqués par SmartGUARD, clôture automatique au retour, confirmation par le client. |
| Important | En mode ligne de code, SmartGUARD ne voit pas le trafic de l'application. | Contrôle à intervalle régulier d'une adresse de l'application, dans les deux modes. |
| Mineur | Les périodes d'indisponibilité du client sont enregistrées dans le service en ligne. | Client informé de l'activation, désactivation possible de son côté. |
| Important | Le masquage des noms de clients est une règle d'interface : techniquement, l'éditeur héberge les données et pourrait y accéder. | Accès techniques journalisés et consultables par le fournisseur ; engagement contractuel de confidentialité de l'éditeur ; chiffrement par le fournisseur écarté pour garder les noms dans les alertes (point 35). |
| Important | La licence SmartGUARD peut être contournée par un utilisateur déterminé (copie, modification). | Accepté : SmartGUARD n'est pas une protection anti-piratage ; la licence sert à facturer et rattacher les installations, pas à empêcher toute fraude. |
| Important | Licence perpétuelle : les revenus sont concentrés à la vente, alors que l'éditeur supporte à vie l'hébergement de l'accès de base et l'envoi des notifications J-30. | Coût unitaire faible (une notification par échéance) ; revenus récurrents portés par l'abonnement et les options ; versions majeures payantes. |
| Mineur | Un fournisseur proche d'un seuil (10 ou 50 serveurs) peut laisser des serveurs non rattachés au portail pour rester dans la formule inférieure. | Accepté : ces serveurs n'ont alors que l'accès de base (notification J-30) ; le portail complet ne les suit pas. |
| Mineur | Certains fournisseurs se contenteront de la licence seule (accès de base) plutôt que de payer l'abonnement (200 000 F CFA par an au minimum). | Accès de base volontairement limité à une notification J-30 ; le portail complet et les options restent réservés à l'abonnement. |
| Important | Confusion entre la licence SmartGUARD et la licence du logiciel du fournisseur, pour les utilisateurs comme pour les développeurs. | Toujours qualifier le mot « licence » dans l'interface, la documentation et le code (note de terminologie, section 1). |
| Important | Un arrêt déclenché par l'éditeur pour un impayé du fournisseur frapperait le client, qui n'est pas partie au litige. | Seul le fournisseur décide des arrêts et blocages ; l'éditeur ne bloque jamais l'application du client (section 4). |
| Important | Légalité des arrêts et blocages : SmartGUARD est un outil légal, mais un arrêt doit reposer sur le contrat entre le fournisseur et son client. | Le fournisseur reste responsable de ses actions configurées ; modèle de clause de suspension recommandé au fournisseur ; actions journalisées. |
| Important | Le fournisseur administre seul les réglages sensibles ; s'il est injoignable, le client ne peut rien ajuster. | Le client garde la consultation, la saisie du code de renouvellement et la réactivation ; procédure de transfert du mot de passe d'administration prévue au contrat entre fournisseur et client. |
| Bloquant | Perte ou vol de la clé privée de l'éditeur : perte = plus aucune licence acceptée par les modules déjà livrés ; vol = licences contrefaites. | Clé conservée hors du dépôt, sauvegardée hors ligne en deux exemplaires (clé USB et coffre) ; en cas de vol, nouvelle paire de clés et nouvelle version du module. |
| Mineur | La mise à jour sans licence permet de garder durablement des serveurs non licenciés. | Accepté (point 39) : état « licence à activer » visible dans l'administration et, plus tard, dans le portail. |
| Mineur | Un client ayant plusieurs fournisseurs utilisant SmartGUARD multiplie les accès. | V1 : un accès par fournisseur ; espace client unique envisagé plus tard. |

## 18. Offre et tarification

Montants en francs CFA (F CFA), hors taxes. Les prix marqués *(proposition)* restent à valider.

### 18.1 Licence SmartGUARD (obligatoire, une par serveur contrôlé)

| Élément | Prix | Statut |
|---|---|---|
| Licence perpétuelle par serveur : module complet, accès de base à la plateforme, notification e-mail à J-30 par échéance, correctifs et mises à jour de sécurité | **150 000** une fois | *(validé le 1er octobre 2026)* |
| Mise à niveau vers une nouvelle version majeure | 75 000 par serveur (50 % du prix de la licence) | *(proposition)* |
| Transfert de la licence vers un nouveau serveur | Gratuit | *(proposition)* |

### 18.2 Abonnement au portail (par compte fournisseur)

Facturé par compte fournisseur, selon le nombre de serveurs **rattachés au portail**. *(validé le 1er octobre 2026)*

| Formule | Serveurs rattachés | Mensuel | Annuel | Économie annuelle |
|---|---|---|---|---|
| Essentiel | jusqu'à 10 | **20 000** | **200 000** | ≈ 17 % |
| Pro | 11 à 50 | **60 000** | **500 000** | ≈ 31 % |
| Entreprise | plus de 50 | **100 000** | **1 000 000** | ≈ 17 % |

- **Contenu commun** : portail complet (toutes les alertes e-mail J-90 à J, événements, module silencieux, récapitulatif hebdomadaire), tableau de bord, calendrier, fiches clients, exports, codes de renouvellement signés, signal des modules ; utilisateurs inclus : **3** (Essentiel), **10** (Pro), **25** (Entreprise) *(proposition)*.
- **Changement de formule** : passage automatique à la formule supérieure dès que le nombre de serveurs rattachés dépasse le seuil, au prorata ; retour à la formule inférieure à l'échéance suivante *(proposition)*.
- Les serveurs avec licence SmartGUARD mais non rattachés au portail ne comptent pas dans le seuil.

### 18.3 Options (nécessitent l'abonnement, prix selon la formule)

Les options suivent la formule de l'abonnement : environ 25 % du prix de la formule pour l'application mobile et 50 % pour l'espace client, avec environ 17 % d'économie à l'année. *(proposition, à valider)*

| Option | Essentiel (mois / an) | Pro (mois / an) | Entreprise (mois / an) |
|---|---|---|---|
| Application mobile du fournisseur (tous les utilisateurs du compte) | 5 000 / 50 000 | 15 000 / 150 000 | 25 000 / 250 000 |
| Espace client : signalement d'incidents, détection de panne confirmée, lien « Signaler un problème », transfert par webhook | 10 000 / 100 000 | 25 000 / 250 000 | 40 000 / 400 000 |
| Utilisateur supplémentaire au-delà des utilisateurs inclus | 2 000 / 20 000 par utilisateur | 2 000 / 20 000 par utilisateur | 2 000 / 20 000 par utilisateur |
| Alertes SMS et WhatsApp (version ultérieure) | Selon consommation | Selon consommation | Selon consommation |

### 18.4 Exemples (formules annuelles)

| Fournisseur | Année 1 | Années suivantes |
|---|---|---|
| 3 serveurs, licence seule | 450 000 | 0 |
| 10 serveurs, Essentiel | 1 500 000 + 200 000 = 1 700 000 | 200 000 |
| 10 serveurs, Essentiel + mobile + espace client | 1 500 000 + 200 000 + 50 000 + 100 000 = 1 850 000 | 350 000 |
| 30 serveurs, Pro | 4 500 000 + 500 000 = 5 000 000 | 500 000 |
| 30 serveurs, Pro + mobile + espace client | 4 500 000 + 500 000 + 150 000 + 250 000 = 5 400 000 | 900 000 |
| 80 serveurs, Entreprise + mobile + espace client | 12 000 000 + 1 000 000 + 250 000 + 400 000 = 13 650 000 | 1 650 000 |

---

## Historique des versions

| Version | Date | Modification |
|---|---|---|
| 1.1 | 28 septembre 2026 | Ajout de la licence par poste et de la date d'arrêt planifiée |
| 1.2 | 30 septembre 2026 | Renommage du projet LicGuard → SmartGUARD ; conversion en Markdown |
| 1.3 | 30 septembre 2026 | Décisions par défaut proposées pour les points de la section 15 (à valider) |
| 1.4 | 1er octobre 2026 | Ajout du portail fournisseur en SaaS (nouvelle section 13) : rôles Fournisseur et Opérateur, rattachement optionnel, alertes, consultation des échéances, code de renouvellement signé ; renumérotation des sections 13 à 16 ; points 10 à 15 validés, points 16 et 17 à préciser |
| 1.5 | 1er octobre 2026 | Précision du modèle : pas de revendeur, l'éditeur exploite le SaaS et le fournisseur paie l'abonnement ; paiements fournisseur/client hors périmètre ; ajout de l'application mobile du fournisseur en option (section 13.8) ; points 17 à 19 |
| 1.6 | 1er octobre 2026 | Ajout de l'espace client et du signalement d'incidents en option (section 13.9) : rôle Contact client, points d'entrée, contexte technique automatique, détection de panne, traitement par le fournisseur ; critères d'acceptation et points 20 à 23 |
| 1.7 | 1er octobre 2026 | Décisions sur l'espace client : intégré dès la V1, signalement simple, canal web indépendant du serveur du client, message pour contrat expiré ; numéro d'urgence, notification prioritaire, relance, aucun délai affiché au client par défaut ; nouvelle section 17 « Risques identifiés et réponses » ; points 21 et 24 à 26 |
| 1.8 | 1er octobre 2026 | Licences suivies au même titre que les contrats (terminologie « échéance », types licence / contrat / abonnement, messages par type, plusieurs échéances par module) ; signalement uniquement manuel, détection automatique abandonnée ; espace client gratuit pour le client et refacturable par le fournisseur hors SmartGUARD ; points 20, 22, 25 validés, points 27 à 29 |
| 1.9 | 1er octobre 2026 | Détection automatique de panne maintenue en option : le client est prévenu et confirme avant toute alerte au fournisseur ; panne non confirmée visible sans alerte ; activation par le fournisseur, client informé ; alerte « module silencieux » inchangée ; points 22, 30 à 32 |
| 1.10 | 1er octobre 2026 | Validation des points 28 (plusieurs échéances par module), 29 (lien « Signaler un problème », désactivé par défaut) et 31 (réglages de la détection de panne) |
| 1.11 | 1er octobre 2026 | Poste = machine serveur hébergeant le logiciel du client et SmartGUARD (section 4 réécrite) ; réglages sensibles administrés par le fournisseur ; Windows 10 et Windows Server 2012 minimum ; heure du serveur du client ; date d'arrêt planifiée optionnelle ; Flutter validé ; conservation six mois après l'échéance ; seuil de détection modifiable ; nouvelle section 18 (fonctionnalités de base et options) ; points 1 à 4, 6 à 9, 19, 23, 31 validés, points 33 et 34 |
| 1.12 | 1er octobre 2026 | « Licence » = licence du logiciel ou des services du fournisseur ; SmartGUARD sans licence propre, vendu par abonnement, module activé avec une clé du compte fournisseur (section 4 réécrite) ; minimum Windows Server 2016 ; confidentialité des noms de clients vis-à-vis de l'éditeur ; points 6, 8, 33, 34 mis à jour, point 35 |
| 1.13 | 1er octobre 2026 | Licence SmartGUARD obligatoire pour chaque serveur de l'application à contrôler, en plus de l'abonnement (section 4 réécrite) ; deux notions de licence toujours qualifiées ; règle : jamais d'arrêt de l'application du client à cause de SmartGUARD ; points 5, 6, 16 (démarche), 17 (modèle), 33 validés ; point 36 ; classement de la section 18 validé |
| 1.14 | 1er octobre 2026 | Arrêts et blocages décidés uniquement par le fournisseur, jamais par l'éditeur (section 4) ; prix estimés : abonnement 13 000 F CFA / mois, licence SmartGUARD 150 000 F CFA par serveur ; points 17 et 36 mis à jour |
| 1.15 | 1er octobre 2026 | Licence SmartGUARD perpétuelle (150 000 F CFA par serveur) incluant l'accès de base et une notification e-mail à J-30 ; abonnement 13 000 F CFA / mois ou 120 000 F CFA / an par compte fournisseur ; correctifs inclus, versions majeures payantes ; section 18 réécrite en offre et tarification avec propositions de prix des options ; points 17, 36 mis à jour, point 37 |
| 1.16 | 1er octobre 2026 | Abonnement en trois formules par compte fournisseur selon les serveurs rattachés : Essentiel 20 000 / mois ou 200 000 / an, Pro 60 000 / mois ou 500 000 / an, Entreprise 100 000 / mois ou 1 000 000 / an ; exemples et point 17 mis à jour |
| 1.17 | 1er octobre 2026 | Prix des options proportionnels aux formules (application mobile ≈ 25 %, espace client ≈ 50 % du prix de la formule) ; utilisateurs inclus par formule (3, 10, 25) ; exemples mis à jour (proposition) |
| 1.17 bis | 1er octobre 2026 | Correction du point 33 (texte écrasé par erreur en v1.14) |
| 1.18 | 1er octobre 2026 | Alignement sur le code existant (module v1.6.0) : date de fin = date d'arrêt avec option « Arrêter l'application à la date de fin » par échéance (section 8 réécrite, point 9) ; ce cahier remplace le v1.6 du dépôt (point 38) ; licence SmartGUARD bloquante à l'installation confirmée |
| 1.19 | 2 octobre 2026 | Licence SmartGUARD : nouvelle installation refusée sans licence valide, mise à jour d'un module existant possible et signalée (point 39) ; licences signées par l'éditeur, clé privée hors du dépôt (point 40) ; licence saisie ou chargée par fichier à l'étape 5 ; liaison au serveur précisée (section 11) ; critère d'acceptation et risques ajoutés |
| 1.20 | 2 octobre 2026 | HTTPS de l'administration en option à l'installation, décochée par défaut, avec certificat auto-signé (point 5, étape 5 de l'assistant) |
