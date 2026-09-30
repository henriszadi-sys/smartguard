# CAHIER DES CHARGES — SmartGUARD

**Module de rappel d'expiration et d'application d'échéance contractuelle**

| Version | Date | Base |
|---|---|---|
| 1.2 | 30 septembre 2026 | LISEZMOI.md et demandes complémentaires (v1.1 du 28 septembre 2026, projet renommé de LicGuard en SmartGUARD) |

*Document de spécification fonctionnelle*

---

## 1. Objet et contexte

SmartGUARD est un module installé sur un serveur pour suivre la date de fin d'un contrat lié à un logiciel. Il avertit les utilisateurs à l'approche de cette date puis peut, à l'échéance ou à une date d'arrêt planifiée, exécuter des actions définies par un technicien : arrêt de l'application ou de services, blocage d'adresses et exécution de scripts.

Le produit comprend un assistant d'installation et de configuration, un service fonctionnant en arrière-plan, une interface web d'administration et un mécanisme de licence par poste.

## 2. Objectifs

- Configurer un module par logiciel surveillé et administrer plusieurs modules indépendants sur un serveur.
- Afficher un avertissement avant la fin du contrat puis appliquer les actions convenues aux dates prévues.
- Permettre le renouvellement du contrat, la réactivation des services et le diagnostic du module.
- Garantir une installation unique de l'application sur chaque poste autorisé, avec une licence individuelle par poste.

## 3. Utilisateurs et rôles

**Administrateur du serveur** — Installe, modifie, désinstalle et diagnostique SmartGUARD. Il choisit le mot de passe d'administration et configure les actions, les ports et la licence ou les postes autorisés.

**Technicien / administrateur SmartGUARD** — Gère les dates de contrat et d'arrêt planifié, l'activation, les actions à exécuter et la réactivation des services après renouvellement.

**Utilisateur de l'application** — Utilise l'application et voit, à l'approche de l'échéance, le message de rappel configuré.

## 4. Licence et installation par poste

- Chaque poste utilisateur doit disposer d'une installation unique de l'application et d'une licence individuelle qui lui est attribuée.
- Une même licence ne peut pas être utilisée simultanément sur plusieurs postes.
- À l'installation ou à l'activation, le système doit identifier le poste et associer sa licence à ce poste.
- L'administration doit permettre de consulter les postes associés aux licences et leur état d'activation.
- Le transfert d'une licence vers un autre poste doit suivre une procédure de désactivation de l'ancien poste puis d'activation du nouveau.
- Le comportement hors ligne, les règles de renouvellement de licence et les limites d'activation restent à préciser avec le commanditaire.

> **Point de cadrage :** cette exigence suppose une licence par poste client. Si la licence visée concerne uniquement le serveur hébergeant SmartGUARD, la règle devra être ajustée.

## 5. Assistant d'installation

L'assistant guide l'administrateur en six étapes :

1. **Logiciel** : nom du logiciel, nom du module (proposé automatiquement) et contact du fournisseur.
2. **Contrat** : dates de début et de fin, délai de rappel (30 jours par défaut), activation et aperçu du message.
3. **Affichage** : mode automatique avec bouton de test, ou intégration par ligne de code.
4. **À l'échéance** : services à arrêter, adresses à bloquer et scripts à exécuter.
5. **Sécurité** : mot de passe administrateur, dossier d'installation et raccourci éventuel.
6. **Récapitulatif** : revue de la configuration et confirmation avant installation.

À la fin, l'assistant affiche l'adresse d'administration et, en mode ligne de code, le code à intégrer à l'application.

## 6. Installation et fonctionnement en service

- Sous Windows Server, l'installation crée un service portant le nom du module, configuré pour démarrer automatiquement avec le serveur.
- Sous Linux, l'installation peut être lancée en ligne de commande. Sans écran, l'assistant fournit une adresse accessible depuis un autre poste.
- Plusieurs modules peuvent fonctionner sur le même serveur, chacun surveillant un logiciel et utilisant son propre port.
- L'installation client de l'application respecte la règle d'un seul poste par licence.

## 7. Rappels et modes d'affichage

Le rappel commence au nombre de jours défini avant la date de fin. Le texte par défaut est : « L'assistance et le support technique à votre logiciel prendra fin dans x jours, veuillez contacter le fournisseur ». Il indique le nombre de jours restants, apparaît en orange puis en rouge à partir de J-7.

- **Automatique** : SmartGUARD se place devant l'application et ajoute le message à ses pages. Le port est configurable ; si SmartGUARD reprend le port courant, l'application doit être déplacée vers un autre port.
- **Ligne de code** : l'application intègre le code fourni par l'assistant pour afficher le message.

## 8. Expiration et arrêt planifié

### Actions à la fin du contrat

La fin du contrat survient à 00:00 le jour de la date de fin. À ce moment, le module exécute les actions de fin de contrat configurées : arrêt et désactivation des services sélectionnés, blocage des adresses avec affichage d'une page « accès suspendu » et exécution des scripts sélectionnés.

### Date d'arrêt planifiée (optionnelle)

Le technicien peut définir une date d'arrêt de l'application, distincte de la date de fin du contrat. Il choisit si l'arrêt concerne l'application, les services liés, ou les deux. Cette date est saisissable dans l'assistant et modifiable dans l'administration.

- À la date définie, SmartGUARD arrête les composants sélectionnés et consigne l'exécution ainsi que toute erreur dans son journal.
- La date d'arrêt planifiée et les actions de fin de contrat sont configurables séparément.
- Sans date d'arrêt planifiée, cette option ne déclenche aucune action.

## 9. Renouvellement et réactivation

L'administrateur peut saisir une nouvelle date de fin dans l'interface d'administration. Après enregistrement, il peut déclencher explicitement l'action « Réactiver les services » pour remettre en service les services concernés.

## 10. Administration, modification et désinstallation

- Connexion à l'interface web avec l'identifiant `admin` et le mot de passe défini à l'installation.
- Consultation et modification des dates, de l'activation, des actions configurées et du journal.
- Affichage de la date d'arrêt planifiée et de l'état des licences/postes associés.
- Depuis l'écran d'accueil : modifier les paramètres, ouvrir l'administration ou désinstaller le module.
- La modification reprend les valeurs existantes ; renommer un module renomme également le service correspondant.
- La désinstallation supprime le service, la règle de pare-feu et les fichiers du module.

## 11. Sécurité

- L'assistant d'installation est accessible uniquement avec le lien secret affiché par son programme.
- Le mot de passe administrateur est stocké sous forme hachée, sans possibilité de récupération en clair.
- L'accès est bloqué après 10 échecs de mot de passe.
- Le recul de l'horloge du serveur ne doit pas repousser l'échéance.
- Un administrateur du serveur peut arrêter le module ; SmartGUARD est un outil de rappel et d'application du contrat, pas une protection anti-piratage.
- La méthode de liaison licence/poste et les protections contre le clonage ou le transfert non autorisé restent à préciser.

## 12. Journaux et diagnostic

- `installation.log` : détail des installations et mises à jour.
- `config.log` : démarrage, erreurs et actions à l'expiration ou à l'arrêt planifié.
- Diagnostic depuis l'assistant : configuration, port, application, état du service et journal d'événements Windows.
- Commande manuelle de diagnostic Windows : exécuter le programme installé avec le paramètre de configuration puis la commande `check`, en invite administrateur.
- Commandes technicien optionnelles : consulter l'état, changer le mot de passe et lancer un diagnostic complet.

## 13. Contraintes techniques connues

- Prise en charge de Windows Server et Linux ; versions minimales à préciser.
- Interface de configuration et d'administration accessible dans un navigateur.
- Un port distinct par module installé sur un serveur.
- Deux modes d'intégration à l'application : proxy automatique ou ligne de code.
- L'antivirus peut mettre le programme en quarantaine ; l'assistant doit signaler le diagnostic et la procédure documentée.

## 14. Critères d'acceptation

- L'assistant présente les six étapes et récapitule les paramètres avant installation.
- Le service est créé et démarre automatiquement sur Windows ; le fonctionnement Linux est accessible selon le mode écran ou sans écran.
- Chaque poste autorisé possède une installation unique et une licence individuelle ; une licence ne peut pas être activée simultanément sur deux postes.
- Le rappel apparaît au délai configuré, indique les jours restants et respecte les couleurs définies.
- Les modes automatique et ligne de code fonctionnent ; le mode automatique propose un test.
- À l'échéance, les actions configurées sont exécutées et journalisées.
- À la date d'arrêt planifiée, l'application et/ou les services sélectionnés sont arrêtés et l'action est journalisée.
- Le renouvellement et la réactivation des services sont possibles depuis l'administration.
- Plusieurs modules coexistent sur un serveur avec des ports distincts.
- Les protections de connexion et les diagnostics prévus sont disponibles.

## 15. Points à préciser avant réalisation

- Le mécanisme de blocage des adresses et d'affichage de la page « accès suspendu ».
- La validation, les droits d'exécution et les tentatives en cas d'échec des scripts.
- La gestion d'un redémarrage serveur autour de l'échéance ou de l'arrêt planifié.
- Les sauvegardes, migrations et restauration des configurations.
- La prise en charge de HTTPS pour l'administration.
- Les versions supportées des systèmes et navigateurs, ainsi que les ressources minimales.
- Le fuseau horaire et le traitement des changements d'heure dans le calcul des échéances.
- La définition exacte d'un « poste » (poste client, terminal virtuel ou poste serveur), le processus d'activation, les transferts, les renouvellements et le mode hors ligne de la licence.
- La relation entre date de fin de contrat et date d'arrêt planifiée si la date d'arrêt est antérieure ou postérieure à la fin du contrat.

---

## Historique des versions

| Version | Date | Modification |
|---|---|---|
| 1.1 | 28 septembre 2026 | Ajout de la licence par poste et de la date d'arrêt planifiée |
| 1.2 | 30 septembre 2026 | Renommage du projet LicGuard → SmartGUARD ; conversion en Markdown |
