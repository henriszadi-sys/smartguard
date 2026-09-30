# SmartGUARD — module de rappel d'expiration (v1.4)

Un seul fichier à lancer. Un assistant graphique demande toutes les informations, puis installe le module comme service sur le serveur.

## Installation sous Windows Server

1. Copiez `Windows\SmartGUARD-Setup.exe` sur le serveur de l'application.
2. **Double-cliquez** dessus, puis acceptez la demande « Autoriser cette application à apporter des modifications ? ».
3. L'assistant s'ouvre dans le navigateur. Laissez la petite fenêtre noire ouverte jusqu'à la fin.
4. Suivez les 6 étapes :

| Étape | Ce qu'on renseigne |
|---|---|
| 1. Logiciel | Nom du logiciel, nom du module (proposé automatiquement, ex. « Kelio-Licence »), contact du fournisseur |
| 2. Contrat | Date de début, date de fin, début du décompte (30 jours par défaut), activation. Un aperçu du message s'affiche |
| 3. Affichage | **Automatique** (le module ajoute le message aux pages de l'application, bouton « Tester ») ou **par une ligne de code** |
| 4. À l'expiration | Services à arrêter (cochés dans la liste des services du serveur), adresses à bloquer, scripts à exécuter |
| 5. Sécurité | Mot de passe administrateur, dossier d'installation, raccourci sur le bureau |
| 6. Récapitulatif | Vérification, puis **Installer** |

À la fin, l'assistant indique l'adresse de la page d'administration et, en mode « ligne de code », la ligne à ajouter dans l'application.

Le programme est copié sous le nom du module (ex. `C:\Program Files\Kelio-Licence\Kelio-Licence.exe`). Un service Windows du même nom est créé, et il démarre automatiquement avec le serveur.

## Installation sous Linux

```
chmod +x smartguard-setup
sudo ./smartguard-setup
```

- **Serveur avec écran** : l'assistant s'ouvre dans le navigateur.
- **Serveur sans écran** : l'adresse de l'assistant s'affiche dans la console (ex. `http://192.168.1.20:8099/?t=…`). Ouvrez-la depuis votre poste.

## Modifier, renommer, désinstaller

Relancez l'assistant, soit `SmartGUARD-Setup.exe`, soit le programme installé. L'écran d'accueil liste les modules installés, avec trois boutons :

- **Modifier** reprend les 6 étapes avec les valeurs actuelles. On peut changer le nom du module : le service est alors renommé.
- **Administration** ouvre la page web de gestion quotidienne (dates, activation, journal).
- **Désinstaller** supprime le service, la règle de pare-feu et les fichiers.

On peut installer plusieurs modules sur un même serveur, un par logiciel surveillé, chacun sur son propre port.

## Mise à jour depuis la version 1.3 (ancien nom LicGuard)

Lancez `SmartGUARD-Setup.exe` (ou `smartguard-setup`) sur le serveur : les modules déjà installés apparaissent sur l'écran d'accueil. Cliquez sur **Modifier** puis **Installer** pour chacun d'eux afin de remplacer le programme.

- Les dates, les actions et le mot de passe sont conservés.
- L'adresse d'administration ne change pas (elle garde `/_licguard/admin`), et la ligne de code déjà ajoutée dans l'application reste valable.
- Les administrateurs connectés devront se reconnecter une fois.

## Fonctionnement

- Le message apparaît à J-30 (réglable) : « L'assistance et le support technique à votre logiciel prendra fin dans x jours, veuillez contacter le fournisseur ». Il est orange, puis rouge à J-7.
- L'expiration a lieu à 00:00 le jour de la date de fin. Les services cochés sont alors arrêtés et désactivés, les adresses sont bloquées (page « accès suspendu ») et les scripts sont exécutés.
- **Renouvellement** : sur la page d'administration, saisissez la nouvelle date de fin, enregistrez, puis cliquez sur **Réactiver les services**.
- **Activer / désactiver** : une case dans l'assistant et dans la page d'administration.

## Mode « Automatique » : à savoir

Le module se place devant l'application. Les utilisateurs passent alors par l'adresse du module (ex. `http://serveur:8080`) :

- si le module prend le port actuel de l'application, déplacez d'abord l'application sur un autre port (ex. Tomcat 8080 → 8081) ;
- sinon, choisissez un nouveau port pour le module et communiquez cette nouvelle adresse aux utilisateurs.

## Connexion à l'administration

L'adresse est affichée à la fin de l'installation (ex. `http://serveur:8090/_smartguard/admin`). Une page de connexion s'ouvre : identifiant **admin**, mot de passe choisi à l'étape 5 de l'assistant. Mot de passe oublié : relancez l'assistant sur le serveur, puis « Modifier » → étape « Sécurité ».

## Sécurité

- L'assistant n'est accessible qu'avec le lien secret affiché dans sa fenêtre.
- Le mot de passe administrateur est stocké chiffré de façon irréversible (haché).
- L'accès est bloqué après 10 échecs de mot de passe.
- Reculer l'horloge du serveur ne permet pas de repousser l'échéance.
- Un administrateur du serveur peut toujours arrêter le module : c'est un outil de rappel et d'application du contrat, pas une protection anti-piratage.

## En cas de problème

- **`installation.log`** (dossier d'installation) : le détail de chaque installation ou mise à jour.
- **`config.log`** : le journal du module (démarrage, erreurs, actions à l'expiration).
- Si le module ne démarre pas, l'assistant affiche un **diagnostic** : configuration, port, application, état du service et journal d'événements Windows.
- Diagnostic manuel (invite de commandes en administrateur) :
  ```
  "C:\Program Files\<Module>\<Module>.exe" -config "C:\Program Files\<Module>\config.json" check
  ```
- Si le programme disparaît du dossier après l'installation, l'antivirus l'a mis en quarantaine. Ajoutez une exclusion pour le dossier d'installation, puis relancez l'assistant.

## Commandes (optionnel, pour les techniciens)

```
Kelio-Licence.exe status         état du décompte
Kelio-Licence.exe set-password   changer le mot de passe
Kelio-Licence.exe check          diagnostic complet
```
