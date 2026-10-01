# PLAN DE DÉVELOPPEMENT — SmartGUARD

| Version | Date | Référence |
|---|---|---|
| 1.3 | 1er octobre 2026 | Cahier des charges v1.18, CLAUDE.md, état des lieux du dépôt (v1.1 : outils et budget ; v1.2 : organisation et rôles ; v1.3 : lots 0 à 5 recalés sur le code existant v1.6.0) |

*Document de pilotage du développement. Le cahier des charges fait foi en cas de désaccord.*

---

## 1. Principes

- **Un lot = un ensemble livrable et testable.** Chaque lot se termine par sa recette (skill `smartguard-recette-livrable`) avant de passer au suivant.
- **Les points encore à valider** (section 16 du cahier des charges) sont développés selon la proposition par défaut, mais isolés dans la configuration pour pouvoir changer sans réécrire le code.
- **Tout comportement lié aux dates est testé avec une horloge simulée** (échéance à 00:00, J-30, J-7, recul d'horloge, redémarrage, seuils).
- **Ordre** : d'abord le module installé chez le client (lots 1 à 5), qui fonctionne seul ; ensuite le portail (lots 6 et 7) ; enfin les options (lot 8).
- **Écart constaté avec le cahier des charges** : on s'arrête, on le signale à Labenie, puis on met à jour le cahier des charges après sa décision.

## 1 bis. Organisation et rôles

| Rôle | Qui | Responsabilités |
|---|---|---|
| **Chef de projet** | Labenie | Priorités, validation des recettes, achats et comptes (hébergement, domaine, certificats, stores, paiement), ARTCI, contrats, relation avec les fournisseurs pilotes |
| **Développement assisté par IA** | Claude | Écriture du code (module Go, portail, application Flutter) et des tests, documentation, procédures d'installation, préparation des procès-verbaux de recette ; travail dans le dépôt Git, lot par lot |
| **Développeur** | Personne physique **à recruter** | Relecture du code de chaque lot ; tests sur Windows 10, Windows Server 2016 et Linux réels ; tests sur téléphones et compilation iPhone (Mac) ; déploiement et exploitation du portail ; détention technique des accès sous l'autorité de Labenie ; continuité entre les sessions |

- **Avant l'arrivée du développeur** : les lots 0 à 4 peuvent avancer (code et tests sous Linux par Claude, validation des résultats par Labenie). Les tests Windows de ces lots sont regroupés et faits dès son arrivée.
- **Indispensable avant le lot 5** : le développeur doit être en poste (installation et service Windows, tests sur machines réelles).
- **Profil recherché** : bonne pratique de Go (ou d'un langage proche) et de Git, administration Windows Server et Linux, notions de PostgreSQL et de déploiement web ; Flutter apprécié pour le lot 8b ; à l'aise pour travailler avec du code produit par une IA et le relire de façon critique.
- Les **charges estimées** de la section 2 restent indicatives : avec cette organisation, l'écriture du code est plus rapide, et la durée réelle dépend surtout du temps de test, de relecture et de mise en place côté humain.

## 2. Vue d'ensemble

**Point de départ** : le dépôt contient déjà un module fonctionnel (v1.6.0, tests au vert) qui couvre l'essentiel des lots 1 à 5 initiaux (voir `Etat_des_lieux_depot_SmartGUARD.md`). Les lots 1 à 5 deviennent des **lots d'adaptation** de ce code au cahier des charges v1.18 ; le code existant est conservé, pas réécrit.

| Lot | Contenu | Composant | Dépend de | Charge estimée* |
|---|---|---|---|---|
| 0 | Mise à niveau du dépôt : cahier v1.18 et CLAUDE.md du projet dans le dépôt, liaison au dépôt GitHub, exécutables et archives sortis de l'historique (publiés en « releases ») | Dépôt | — | 1 j |
| 1 | Échéances multiples (licence, contrat, abonnement) avec migration de l'ancienne configuration à une échéance | Module | 0 | 5 j |
| 2 | Rappels et bandeau pour plusieurs échéances, messages par type, lien « Signaler un problème » (désactivé) | Module | 1 | 3 j |
| 3 | Option d'arrêt et actions propres à chaque échéance, exécution unique par échéance, réactivation | Module | 1 | 3 j |
| 4 | Licence SmartGUARD bloquante à l'installation (existant : clés signées, liaison au serveur) | Module | 0 | 3 j |
| 5 | Rôles technicien du fournisseur / accès client, contrôle Windows 10 / Server 2016, version 1.7.0 | Module | 1 à 4 | 6 j |
| 6 | Portail : comptes, rôles, accès de base, abonnements et droits | Portail SaaS | 4 | 10 j |
| 7 | Portail : enrôlement, signal, alertes, tableau de bord, codes de renouvellement | Portail SaaS | 5, 6 | 15 j |
| 8a | Espace client : signalements, détection de panne confirmée par le client | Portail + module | 7 | 12 j |
| 8b | Application mobile du fournisseur (Flutter) | Mobile | 7 | 15 j |
| | **Total** | | | **≈ 73 jours de travail** |

\* Estimation indicative, hors recette et hors imprévus. Les lots 2, 3 et 4 peuvent avancer en parallèle une fois le lot 1 terminé.

**Jalons**

- **Jalon A (lots 0 à 5)** : module SmartGUARD v1.7.0 conforme au cahier v1.18, installable et utilisable sans portail. Peut être montré à un premier fournisseur pilote.
- **Jalon B (lots 6 et 7)** : portail en ligne, accès de base et abonnements. Mise en service commerciale (après la vérification ARTCI).
- **Jalon C (lot 8)** : options espace client et application mobile.

## 3. Détail des lots

Les critères (C1 à C31) renvoient à la liste numérotée de la section 15 du cahier des charges, dans l'ordre où ils y apparaissent.

### Lot 0 — Mise à niveau du dépôt

- **Contenu** : remplacer dans le dépôt le cahier des charges v1.6 et le `CLAUDE.md` par les versions du projet (v1.18), en conservant les informations techniques propres au code existant (bibliothèque de service, hachage PBKDF2, fichiers de configuration, outil `sgkeys`, exceptions LicGuard) ; relier le dépôt local à `github.com/henriszadi-sys/smartguard` ; retirer les exécutables et archives de l'historique courant (`.gitignore`) et les publier comme « releases » GitHub.
- **Livrable** : dépôt à jour et synchronisé avec GitHub, tests toujours au vert.

### Lot 1 — Échéances multiples

- **Existant** : une seule échéance (`end_date`, `stop_on_end`), calcul à 00:00 heure du serveur, J-7, anti-recul d'horloge, horloge injectable, migration de `stop_date`.
- **À faire** : liste d'échéances par module (type licence, contrat de support / maintenance, abonnement ; dates ; option d'arrêt ; délai de rappel ; messages) ; migration automatique d'une configuration v1.6 à une échéance ; état persistant par échéance ; adaptation de l'assistant (étape 2) et de l'administration.
- **Critères couverts** : C18, base de C8 et C10.
- **Tests** : migration v1.6 → v1.7 sans perte ; plusieurs échéances indépendantes ; échéance à 00:00 ; J-30 et J-7 ; recul d'horloge ; fuseau du serveur.

### Lot 2 — Rappels pour plusieurs échéances

- **Existant** : bandeau orange / rouge à J-7, proxy automatique, mode ligne de code, page « accès suspendu ».
- **À faire** : affichage de plusieurs échéances (la plus proche en premier), textes par défaut selon le type, lien « Signaler un problème » préparé et désactivé par défaut.
- **Critères couverts** : C8, C9.

### Lot 3 — Arrêt et actions par échéance

- **Existant** : services, blocage d'adresses, scripts (3 tentatives, délai), exécution unique et rattrapage au démarrage, réactivation explicite.
- **À faire** : actions et option d'arrêt propres à chaque échéance ; exécution unique suivie par échéance ; affichage de l'échéance à l'origine d'un arrêt ; réactivation par échéance.
- **Critères couverts** : C10, C11, C12, C4 (côté module).

### Lot 4 — Licence SmartGUARD bloquante

- **Existant** : clés signées Ed25519 (`SGL1.…`), outil `sgkeys`, liaison au serveur hors ligne, transfert, détection d'installation copiée ; licence informative.
- **À faire** : l'assistant refuse d'installer sans licence valide (saisie de la clé à l'installation) ; une fois installé, le module ne s'arrête jamais à cause de la licence ; activation par fichier hors ligne si nécessaire ; renseigner `public.key` (mode signé) avant la première livraison.
- **Critères couverts** : C3.

### Lot 5 — Rôles, systèmes et version 1.7.0

- **Existant** : compte `admin`, blocage après 10 échecs, HTTPS possible, plusieurs modules par serveur, service Windows / systemd.
- **À faire** : distinction technicien du fournisseur (réglages réservés : page « accès suspendu », scripts, redémarrage, sauvegardes) et accès client limité (consultation, code de renouvellement, réactivation) ; contrôle des systèmes minimaux (Windows 10 / Server 2016) ; passage en version 1.7.0, documentation `LISEZMOI.md` ; tests d'installation sur machines réelles par le développeur.
- **Critères couverts** : C1, C2, C6, C7, C13, C14.
- **Fin du jalon A** : recette complète du module v1.7.0.

### Lot 6 — Portail : comptes, accès et abonnements

- **Contenu** : portail Go avec PostgreSQL et API REST ; comptes fournisseurs et utilisateurs (gestionnaire, commercial, lecture seule), double authentification ; accès de base inclus dans la licence (saisie des clients et échéances, notification J-30) ; formules Essentiel, Pro, Entreprise selon les serveurs rattachés ; options activées par compte ; droits vérifiés côté serveur ; isolation stricte entre fournisseurs ; administration éditeur avec identifiants seulement et journal d'audit ; abonnement expiré = retour à l'accès de base.
- **Critères couverts** : C5, C22, C17 (partie accès de base), C16 (partie cloisonnement).
- **Tests** : cloisonnement sur chaque requête ; changement de formule au passage d'un seuil ; abonnement expiré ; l'éditeur ne voit jamais un nom de client.

### Lot 7 — Portail : suivi, alertes et renouvellement

- **Contenu** : clés d'enrôlement ; signal du module toutes les 6 h (HTTPS sortant, file d'attente en cas d'échec) ; détachement ; saisie manuelle et import du fichier d'état ; alertes e-mail calculées par le portail (J-90 à J, événements, module silencieux, récapitulatif hebdomadaire) ; tableau de bord, filtres, calendrier, fiches clients, exports ; codes de renouvellement signés (Ed25519) et leur vérification par le module ; garantie qu'aucune fonction de l'éditeur ne déclenche d'arrêt chez le client.
- **Critères couverts** : C4, C15, C16, C17, C19, C20, C21.
- **Tests** : alertes envoyées avec module éteint ; module silencieux après 48 h ; code accepté par le bon module, refusé ailleurs ou modifié ; export conforme.
- **Fin du jalon B** : mise en service possible après la vérification ARTCI (point 16).

### Lot 8a — Espace client et détection de panne

- **Contenu** : espace client web hébergé dans le SaaS ; invitation des contacts ; signalements (gravité, cause présumée, pièces jointes, contexte technique) ; statuts et messagerie ; message « support expiré » et mention « hors contrat » ; numéro d'urgence ; relance du fournisseur ; webhook ; détection de panne (contrôle d'une adresse, seuil de 5 minutes modifiable, plages de maintenance) avec confirmation obligatoire du client ; conservation six mois après l'échéance.
- **Critères couverts** : C24 à C31.
- **Tests** : signalement avec serveur du client éteint ; aucune alerte au fournisseur sans action du client ; panne « détectée, non confirmée » visible sans alerte ; clôture automatique ; arrêt SmartGUARD jamais détecté comme panne.

### Lot 8b — Application mobile du fournisseur

- **Contenu** : application Flutter (Android d'abord, puis iOS) ; notifications (Firebase Cloud Messaging et service Apple) ; consultation des contrats, licences et modules ; mêmes rôles que le portail ; cache chiffré ; déconnexion à distance.
- **Critères couverts** : C23.
- **Prérequis** : comptes développeur Google Play et Apple.

## 4. Outils, logiciels, bases de données et services

Les noms de produits sont des exemples ; le choix final reste ouvert. La colonne **Budget prévu** est à remplir par Labenie (en F CFA). Les coûts indicatifs sont des ordres de grandeur à vérifier au moment de l'achat.

### 4.1 Développement

| Élément | Usage | Nécessaire pour | Coût indicatif | Budget prévu | Responsable |
|---|---|---|---|---|---|
| Go (langage) | Module et portail | Lot 1 | Gratuit | | Développeur |
| Git + GitHub ou GitLab | Dépôt du code, historique, travail en équipe | Lot 1 | Gratuit (petite équipe) | | Labenie / développeur |
| VS Code (ou GoLand) | Éditeur de code | Lot 1 | Gratuit (GoLand payant) | | Développeur |
| Intégration continue (GitHub Actions ou équivalent) | Compilation et tests automatiques | Lot 1 | Gratuit dans une limite d'usage | | Développeur |
| Flutter + Android Studio | Application mobile Android | Lot 8b | Gratuit | | Développeur |
| Mac avec Xcode (achat ou location en ligne) | **Obligatoire** pour compiler l'application iPhone | Lot 8b | À chiffrer | | Labenie |

### 4.2 Bases de données et stockage

| Élément | Usage | Nécessaire pour | Coût indicatif | Budget prévu | Responsable |
|---|---|---|---|---|---|
| Aucune base de données dans le module | Configuration dans un fichier JSON sur le serveur du client (voulu : rien à installer chez le client) | — | — | — | — |
| PostgreSQL (auto-hébergé ou service géré : Supabase, Neon…) | Base du portail : comptes, clients, échéances, licences, abonnements, signalements | Lot 6 | Gratuit à mensuel selon l'offre | | Labenie |
| Stockage de fichiers compatible S3 | Pièces jointes de l'espace client, sauvegardes | Lot 8a | Mensuel, selon volume | | Labenie |

### 4.3 Hébergement du portail

| Élément | Usage | Nécessaire pour | Coût indicatif | Budget prévu | Responsable |
|---|---|---|---|---|---|
| Serveur (VPS ou cloud) | Portail SaaS ; localisation selon la vérification ARTCI | Lot 6 | Mensuel | | Labenie |
| Nom de domaine (ex. smartguard.ci) | Adresse du portail et des e-mails | Lot 6 | Annuel | | Labenie |
| Certificat HTTPS (Let's Encrypt) | Connexion sécurisée au portail | Lot 6 | Gratuit | | Développeur |
| Sauvegardes automatiques | Base et fichiers du portail | Lot 6 | Inclus ou mensuel | | Développeur |
| Surveillance (disponibilité et erreurs) | Alerte si le portail tombe, suivi des erreurs | Lot 7 | Gratuit à mensuel | | Développeur |

### 4.4 Services externes

| Élément | Usage | Nécessaire pour | Coût indicatif | Budget prévu | Responsable |
|---|---|---|---|---|---|
| Envoi d'e-mails (Brevo, SendGrid…) | Alertes d'échéance, notification J-30, espace client | Lot 7 | Gratuit à mensuel selon volume | | Labenie |
| Firebase Cloud Messaging + service Apple (APNs) | Notifications sur téléphone | Lot 8b | Gratuit | | Développeur |
| Paiement des abonnements (Mobile Money, carte ; agrégateur local à choisir, ex. CinetPay, PayDunya — à vérifier) | Encaisser licences et abonnements des fournisseurs | Avant le jalon B | Commission par transaction | | Labenie |
| SMS et WhatsApp (Twilio ou opérateur local) | Alertes SMS / WhatsApp | Version ultérieure | Par message | | Labenie |

### 4.5 Tests

| Élément | Usage | Nécessaire pour | Coût indicatif | Budget prévu | Responsable |
|---|---|---|---|---|---|
| VirtualBox ou Hyper-V | Machines virtuelles de test | Lot 3 | Gratuit | | Développeur |
| Windows 10 et Windows Server 2016 (version d'évaluation Microsoft) | Tests d'installation et de fonctionnement | Lots 3 à 5 | Gratuit (évaluation) ; licence si usage prolongé | | Développeur |
| Ubuntu | Tests Linux / systemd | Lots 3 à 5 | Gratuit | | Développeur |
| Téléphone Android et iPhone | Tests de l'application mobile | Lot 8b | Achat éventuel | | Labenie |

### 4.6 Distribution et démarches

| Élément | Usage | Nécessaire pour | Coût indicatif | Budget prévu | Responsable |
|---|---|---|---|---|---|
| Certificat de signature de code Windows | Limite les blocages antivirus et l'avertissement « éditeur inconnu » | Lot 5 | Annuel | | Labenie |
| Compte Google Play | Publication Android | Lot 8b | Environ 25 $, paiement unique | | Labenie |
| Compte Apple Developer | Publication iPhone | Lot 8b | Environ 99 $ par an | | Labenie |
| Déclaration ou autorisation ARTCI (loi n° 2013-450) | Mise en ligne du portail | Avant le jalon B | À vérifier | | Labenie |
| Conditions générales et contrat de licence SmartGUARD ; modèle de clause de suspension pour les fournisseurs | Vente des licences et abonnements ; encadrement des blocages | Avant le jalon B | Honoraires éventuels | | Labenie |

### 4.7 Ordre d'acquisition

1. **Lot 1** : Go et un dépôt Git (rien à acheter).
2. **Lot 3** : machines virtuelles de test.
3. **Lot 5** : certificat de signature de code.
4. **Lots 6 et 7** : hébergement, PostgreSQL, nom de domaine, service d'e-mails ; démarches ARTCI, paiement des abonnements, contrats.
5. **Lot 8** : stockage de fichiers, comptes Google Play et Apple, Mac, téléphones de test.

## 5. Points encore ouverts qui touchent le développement

| Point | Lot concerné | Traitement en attendant |
|---|---|---|
| Prix des options, utilisateurs inclus, changement de formule (section 18) | 6 | Paramètres modifiables dans l'administration éditeur |
| Ressources minimales par module | 5 | Mesurer pendant la recette du jalon A |
| Protection des données / ARTCI (point 16) | 6, 7, 8a | Hébergement et durées de conservation paramétrables |

## 6. Suivi

| Lot | Statut | Début | Fin | Recette |
|---|---|---|---|---|
| Recrutement du développeur | À faire (avant le lot 5) | | | |
| 0 | À démarrer | | | |
| 1 | À faire | | | |
| 2 | À faire | | | |
| 3 | À faire | | | |
| 4 | À faire | | | |
| 5 | À faire | | | |
| 6 | À faire | | | |
| 7 | À faire | | | |
| 8a | À faire | | | |
| 8b | À faire | | | |
