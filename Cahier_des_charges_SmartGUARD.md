# CAHIER DES CHARGES — SmartGUARD

**Module de rappel d'expiration et d'application d'échéance contractuelle**

| Version | Date | Base |
|---|---|---|
| 1.5 | 30 septembre 2026 | LISEZMOI.md et demandes complémentaires (v1.1 du 28 septembre 2026, projet renommé de LicGuard en SmartGUARD) ; date de fin = date d'arrêt ; licence par poste = serveur |

*Document de spécification fonctionnelle*

---

## 1. Objet et contexte

SmartGUARD est un module installé sur un serveur pour suivre la date de fin d'un contrat lié à un logiciel. Il avertit les utilisateurs à l'approche de cette date puis peut, à cette date de fin, exécuter des actions définies par un technicien : arrêt de l'application ou de services, blocage d'adresses et exécution de scripts.

Le produit comprend un assistant d'installation et de configuration, un service fonctionnant en arrière-plan, une interface web d'administration et un mécanisme de licence par poste.

## 2. Objectifs

- Configurer un module par logiciel surveillé et administrer plusieurs modules indépendants sur un serveur.
- Afficher un avertissement avant la fin du contrat puis appliquer les actions convenues aux dates prévues.
- Permettre le renouvellement du contrat, la réactivation des services et le diagnostic du module.
- Garantir une installation unique de l'application sur chaque poste autorisé, avec une licence individuelle par poste.

## 3. Utilisateurs et rôles

**Administrateur du serveur** — Installe, modifie, désinstalle et diagnostique SmartGUARD. Il choisit le mot de passe d'administration et configure les actions, les ports et la licence ou les postes autorisés.

**Technicien / administrateur SmartGUARD** — Gère les dates de contrat, l'option d'arrêt à la date de fin, l'activation, les actions à exécuter et la réactivation des services après renouvellement.

**Utilisateur de l'application** — Utilise l'application et voit, à l'approche de l'échéance, le message de rappel configuré.

## 4. Licence et installation par poste

- Un **poste** est le serveur qui héberge le module, identifié par un identifiant machine (`MachineGuid` sous Windows, `/etc/machine-id` sous Linux), conservé sous forme hachée (16 caractères).
- Le module dispose d'une licence individuelle, dont la clé a la forme `SGRD-XXXX-XXXX-XXXX-XXXX` (le dernier groupe est une somme de contrôle). Une même licence ne doit pas être utilisée simultanément sur plusieurs postes.
- À l'activation, le système identifie le poste et lie la licence à ce poste (fichier `config.license.json`, vérifié hors ligne).
- L'administration affiche le poste, la clé masquée, la date d'activation et l'état : non activée, activée sur ce poste, liée à un autre poste (installation copiée). La clé complète n'est jamais affichée ni journalisée.
- Transfert vers un autre poste : désactivation sur l'ancien poste, puis activation sur le nouveau. Activer sur un autre poste sans désactivation préalable est refusé.
- Activation, désactivation et état sont aussi disponibles en ligne de commande (`license`).
- La licence est informative : elle ne bloque ni le décompte ni l'arrêt à la date de fin.
- **Limite connue** : sans serveur central, SmartGUARD ne peut pas détecter qu'une même clé est activée sur deux serveurs distincts ; il détecte seulement une installation copiée. Le mode hors ligne, les règles de renouvellement de licence et les limites d'activation restent à préciser avec le commanditaire.

## 5. Assistant d'installation

L'assistant guide l'administrateur en six étapes :

1. **Logiciel** : nom du logiciel, nom du module (proposé automatiquement) et contact du fournisseur.
2. **Contrat** : dates de début et de fin, option d'arrêt à la date de fin, délai de rappel (30 jours par défaut), activation et aperçu du message.
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

## 8. Expiration et arrêt

### Fin du contrat = date d'arrêt

Il n'existe qu'une seule date : la date de fin du contrat ou de la licence. Elle est aussi la date d'arrêt de l'application. Elle est saisissable dans l'assistant et modifiable dans l'administration.

La fin survient à 00:00 le jour de la date de fin. À ce moment, le message de rappel passe à l'état « expiré ».

### Option « Arrêter l'application à la date de fin »

Case à cocher, décochée tant que le technicien ne l'a pas activée.

- **Cochée** : à 00:00 le jour de la date de fin, SmartGUARD exécute une seule fois les actions configurées (arrêt et désactivation des services sélectionnés, blocage des adresses avec page « accès suspendu », exécution des scripts) et consigne l'exécution ainsi que toute erreur dans son journal.
- **Décochée** : seul le message passe à « expiré » ; aucune action n'est jamais déclenchée automatiquement.
- Le recul de l'horloge du serveur ne repousse pas l'arrêt.
- Migration : dans une configuration issue d'une version antérieure, l'ancienne date d'arrêt (`stop_date`) est reprise comme date de fin et l'option est cochée.

## 9. Renouvellement et réactivation

L'administrateur peut saisir une nouvelle date de fin dans l'interface d'administration. Après enregistrement, il peut déclencher explicitement l'action « Réactiver les services » pour remettre en service les services concernés.

## 10. Administration, modification et désinstallation

- Connexion à l'interface web avec l'identifiant `admin` et le mot de passe défini à l'installation.
- Consultation et modification des dates, de l'activation, des actions configurées et du journal.
- Affichage de la date de fin, de l'option d'arrêt et de l'état des licences/postes associés.
- Depuis l'écran d'accueil : modifier les paramètres, ouvrir l'administration ou désinstaller le module.
- La modification reprend les valeurs existantes ; renommer un module renomme également le service correspondant.
- La désinstallation supprime le service, la règle de pare-feu et les fichiers du module.

## 11. Sécurité

- L'assistant d'installation est accessible uniquement avec le lien secret affiché par son programme.
- Le mot de passe administrateur est stocké sous forme hachée, sans possibilité de récupération en clair.
- L'accès est bloqué après 10 échecs de mot de passe.
- Le recul de l'horloge du serveur ne doit pas repousser l'échéance.
- Un administrateur du serveur peut arrêter le module ; SmartGUARD est un outil de rappel et d'application du contrat, pas une protection anti-piratage.
- La liaison licence/poste est locale et hors ligne : elle détecte une installation copiée mais ne protège pas contre un contournement volontaire (section 4).

## 12. Journaux et diagnostic

- `installation.log` : détail des installations et mises à jour.
- `config.log` : démarrage, erreurs et actions à l'expiration ou à l'arrêt.
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
- À la fin du contrat, le message passe à « expiré » ; sans l'option d'arrêt, aucune action n'est exécutée.
- À la date de fin, si l'option d'arrêt est cochée, les actions configurées (services, adresses bloquées, scripts) sont exécutées une seule fois et journalisées.
- Le renouvellement et la réactivation des services sont possibles depuis l'administration.
- Plusieurs modules coexistent sur un serveur avec des ports distincts.
- Les protections de connexion et les diagnostics prévus sont disponibles.

## 15. Points à préciser avant réalisation

- Le mécanisme de blocage des adresses et d'affichage de la page « accès suspendu ».
- La validation, les droits d'exécution et les tentatives en cas d'échec des scripts.
- La gestion d'un redémarrage serveur autour de l'échéance ou de l'arrêt.
- Les sauvegardes, migrations et restauration des configurations.
- La prise en charge de HTTPS pour l'administration.
- Les versions supportées des systèmes et navigateurs, ainsi que les ressources minimales.
- Le fuseau horaire et le traitement des changements d'heure dans le calcul des échéances.
- ~~La définition exacte d'un « poste »~~ : décidé en v1.5, le poste est le serveur qui héberge le module. Restent à préciser : l'émission et le contrôle des clés par le fournisseur, les renouvellements, les limites d'activation et un éventuel contrôle en ligne.
- ~~La relation entre date de fin de contrat et date d'arrêt planifiée~~ : décidé en v1.4, une seule date ; l'arrêt à cette date est une option.

---

## Historique des versions

| Version | Date | Modification |
|---|---|---|
| 1.1 | 28 septembre 2026 | Ajout de la licence par poste et de la date d'arrêt planifiée |
| 1.2 | 30 septembre 2026 | Renommage du projet LicGuard → SmartGUARD ; conversion en Markdown |
| 1.3.1 | 30 septembre 2026 | La fin du contrat n'arrête plus l'application ; les actions ne se déclenchent qu'à la date d'arrêt optionnelle |
| 1.4 | 30 septembre 2026 | La date de fin du contrat est la date d'arrêt : suppression de la date d'arrêt distincte, remplacée par l'option « Arrêter l'application à la date de fin » ; reprise de `stop_date` à la migration |
| 1.5 | 30 septembre 2026 | Licence par poste implémentée : poste = serveur, liaison locale hors ligne (`config.license.json`), activation / désactivation / transfert dans l'administration et en ligne de commande, détection d'une installation copiée |
