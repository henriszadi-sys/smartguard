# SmartGUARD — module de rappel d'échéance (v1.10)

SmartGUARD suit, sur le serveur d'un client, la date de fin d'une **licence** ou d'un **contrat** (support, maintenance, abonnement) d'un logiciel du fournisseur. Il prévient les utilisateurs avant la date de fin puis, si le fournisseur l'a prévu, applique les actions configurées à cette date.

Un seul fichier à lancer : un assistant graphique demande toutes les informations, puis installe le module comme service sur le serveur.

## Systèmes pris en charge

- **Windows 10** et **Windows Server 2016** ou plus récents.
- **Linux avec systemd** (Ubuntu 20.04+, Debian 11+, RHEL 8+).

L'assistant et la commande `check` vérifient le système ; sur un système non pris en charge, l'installation est refusée.

## Licence SmartGUARD

- **Une licence par serveur.** Elle couvre tous les modules SmartGUARD installés sur ce serveur.
- **Obligatoire pour installer un nouveau module** : à l'étape « Sécurité », saisissez la clé fournie par l'éditeur (forme `SGL1.…`, longue : copiez-la en entier) ou chargez le fichier de licence. Si le serveur a déjà une licence active, rien n'est demandé.
- **Mise à jour** d'un module déjà installé : possible sans licence ; l'assistant et l'administration indiquent alors « licence à activer ».
- **Une fois installé, le module ne s'arrête jamais à cause de la licence.**
- Seules les clés émises par l'éditeur sont acceptées. **Transfert** vers un autre serveur : désactivez la licence sur l'ancien (administration ou commande `license deactivate`), puis activez-la sur le nouveau.

## Installation sous Windows

1. Copiez `SmartGUARD-Setup.exe` sur le serveur de l'application.
2. **Double-cliquez** dessus, puis acceptez la demande « Autoriser cette application à apporter des modifications ? ».
3. L'assistant s'ouvre dans le navigateur. Laissez la petite fenêtre noire ouverte jusqu'à la fin.
4. Suivez les 6 étapes :

| Étape | Ce qu'on renseigne |
|---|---|
| 1. Logiciel | Nom du logiciel, nom du module (proposé automatiquement, ex. « Kelio-Licence »), contact du fournisseur |
| 2. Échéance | Type (contrat de support, licence, abonnement) et libellé, date de début, date de fin (avec l'option « Arrêter l'application à la date de fin »), début du décompte (30 jours par défaut), activation. Un aperçu du message s'affiche. Les autres échéances s'ajoutent ensuite dans l'administration |
| 3. Affichage | **Automatique** (le module ajoute le message aux pages de l'application, bouton « Tester ») ou **par une ligne de code** |
| 4. À l'arrêt | Services à arrêter (cochés dans la liste des services du serveur), adresses à bloquer, scripts à exécuter |
| 5. Sécurité | Licence SmartGUARD du serveur (si elle n'est pas encore active), mot de passe du technicien, dossier d'installation, raccourci sur le bureau |
| 6. Récapitulatif | Vérification, puis **Installer** |

À la fin, l'assistant indique l'adresse de la page d'administration et, en mode « ligne de code », la ligne à ajouter dans l'application.

Le programme est copié sous le nom du module (ex. `C:\Program Files\Kelio-Licence\Kelio-Licence.exe`). Un service Windows du même nom est créé ; il démarre automatiquement avec le serveur.

## Installation sous Linux

```
chmod +x smartguard-setup
sudo ./smartguard-setup
```

- **Serveur avec écran** : l'assistant s'ouvre dans le navigateur.
- **Serveur sans écran** : l'adresse de l'assistant s'affiche dans la console (ex. `http://192.168.1.20:8099/?t=…`). Ouvrez-la depuis votre poste.

## Modifier, renommer, désinstaller

Relancez l'assistant (`SmartGUARD-Setup.exe` ou le programme installé). L'écran d'accueil liste les modules installés :

- **Modifier** reprend les 6 étapes avec les valeurs actuelles (la première échéance ; les autres sont conservées). Changer le nom du module renomme le service. **Une sauvegarde de la configuration est faite automatiquement** avant chaque mise à jour, dans le dossier `sauvegardes` du module (10 dernières conservées).
- **Administration** ouvre la page web de gestion.
- **Désinstaller** supprime le service, la règle de pare-feu et les fichiers du module ; la licence du serveur reste active pour les autres modules.

Plusieurs modules peuvent être installés sur un même serveur, un par logiciel surveillé, chacun sur son propre port.

## Échéances

Un module peut suivre **plusieurs échéances** d'un même logiciel, par exemple sa licence et son contrat de support. Chacune a son type, ses dates, son délai de rappel, ses messages et ses actions.

- Le message apparaît à J-30 (réglable), en orange, puis **en rouge à partir de J-7**. Le texte par défaut dépend du type d'échéance et reste modifiable.
- Si plusieurs échéances approchent, le bandeau montre la plus urgente et rappelle les autres sur une seconde ligne.
- L'échéance tombe à **00:00 le jour de la date de fin**, à l'heure du serveur : le message passe à « expiré ».
- La date de fin est aussi la **date d'arrêt** si l'option « Arrêter l'application à la date de fin » est cochée pour cette échéance : à 00:00 ce jour-là, ses services sont arrêtés et désactivés, ses adresses sont bloquées (page « accès suspendu », qui nomme l'échéance concernée) et ses scripts sont exécutés, **une seule fois**. Option décochée : aucune action, seul le message change.
- Reculer l'horloge du serveur ne repousse jamais une échéance. Si le serveur était éteint à la date de fin, les actions manquées sont exécutées une fois au démarrage.

## Renouvellement et réactivation

1. Dans l'administration, saisissez la nouvelle date de fin de l'échéance et **Enregistrez**.
2. Cliquez sur **Réactiver les services de cette échéance** (ou **Réactiver les services** pour toutes les échéances renouvelées).

Seuls les services arrêtés par SmartGUARD sont relancés, et jamais tant que l'échéance est encore arrêtée. La réactivation n'est jamais automatique.

## Administration : technicien et accès client

L'adresse est affichée à la fin de l'installation (ex. `http://serveur:8090/_smartguard/admin`).

- **Technicien du fournisseur** : identifiant **admin**, mot de passe choisi à l'étape 5. Il règle tout : échéances, actions, lien « Signaler un problème », licence, sauvegardes, accès client.
- **Accès client** (désactivé par défaut) : le technicien l'active dans la section « Accès client » en choisissant un mot de passe différent du sien (identifiant **client** par défaut). L'administrateur du serveur du client peut alors **consulter** l'administration et **réactiver les services**, sans pouvoir modifier aucun réglage.

Mot de passe du technicien oublié : relancez l'assistant sur le serveur, « Modifier » → étape « Sécurité », ou utilisez la commande `set-password`.

## Sauvegarde de la configuration

Dans l'administration (technicien), section « Sauvegarde de la configuration » :

- **Exporter** télécharge un fichier `.smartguard.json` avec les échéances et les réglages, **sans aucun mot de passe**.
- **Importer** remplace les échéances et réglages par ceux du fichier, en gardant le port, l'adresse de l'application et les mots de passe du module.

## Lien « Signaler un problème »

Désactivé par défaut. Le technicien peut l'activer en indiquant l'adresse de signalement (espace client ou support du fournisseur) : le lien apparaît dans les pages de l'application, dans le bandeau et sur la page « accès suspendu ».

## Mode « Automatique » : à savoir

Le module se place devant l'application. Les utilisateurs passent alors par l'adresse du module (ex. `http://serveur:8080`) :

- si le module prend le port actuel de l'application, déplacez d'abord l'application sur un autre port (ex. Tomcat 8080 → 8081) ;
- sinon, choisissez un nouveau port pour le module et communiquez cette nouvelle adresse aux utilisateurs.

## Mise à jour depuis une version précédente

Lancez la nouvelle version de l'assistant sur le serveur : les modules installés apparaissent sur l'écran d'accueil. Cliquez sur **Modifier** puis **Installer** pour chacun d'eux.

- Les dates, les actions et les mots de passe sont conservés ; la configuration est sauvegardée avant la mise à jour.
- Une configuration v1.6 (une seule échéance) devient une échéance « Contrat de support », sans perte ; les actions déjà exécutées ne sont pas refaites.
- Une licence déjà activée par un module devient la licence du serveur.
- Les adresses d'administration ne changent pas ; la ligne de code déjà ajoutée dans l'application reste valable.

## Sécurité

- L'assistant n'est accessible qu'avec le lien secret affiché dans sa fenêtre.
- Les mots de passe sont stockés hachés, de façon irréversible.
- L'accès est bloqué après 10 échecs de mot de passe.
- Les clés de licence et les mots de passe n'apparaissent jamais en clair dans les journaux.
- Un administrateur du serveur peut toujours arrêter le module : c'est un outil de rappel et d'application du contrat, pas une protection anti-piratage.

## En cas de problème

- **`installation.log`** (dossier d'installation) : le détail de chaque installation ou mise à jour.
- **`config.log`** : le journal du module (démarrage, connexions, modifications, actions à la date de fin, réactivations).
- Si le module ne démarre pas, l'assistant affiche un **diagnostic** : système, configuration, port, application, état du service et journal d'événements Windows.
- Diagnostic manuel (invite de commandes en administrateur) :
  ```
  "C:\Program Files\<Module>\<Module>.exe" -config "C:\Program Files\<Module>\config.json" check
  ```
- Si le programme disparaît du dossier après l'installation, l'antivirus l'a mis en quarantaine. Ajoutez une exclusion pour le dossier d'installation, puis relancez l'assistant.

## Commandes (pour les techniciens)

```
Kelio-Licence.exe status                    état de chaque échéance
Kelio-Licence.exe check                     diagnostic complet (système compris)
Kelio-Licence.exe set-password              mot de passe du technicien
Kelio-Licence.exe set-password client       mot de passe de l'accès client (l'active)
Kelio-Licence.exe license                   état de la licence SmartGUARD du serveur
Kelio-Licence.exe license activate <clé>    activer la licence sur ce serveur
Kelio-Licence.exe license deactivate        désactiver (première étape d'un transfert)
Kelio-Licence.exe install                   installer le service (licence active exigée)
```

Sous Windows, ajoutez `-config "<dossier>\config.json"` si la commande n'est pas lancée depuis le dossier du module.
