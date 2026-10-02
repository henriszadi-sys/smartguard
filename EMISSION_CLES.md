# Émission des licences SmartGUARD (éditeur)

Ce document est destiné à l'éditeur de SmartGUARD. Il ne doit pas être livré aux fournisseurs ni aux clients.

## Principe

Les licences SmartGUARD sont signées avec Ed25519. L'éditeur garde la **clé privée** ; le module installé chez le client ne contient que la **clé publique** (`internal/license/public.key`) et refuse toute clé qui n'a pas été émise avec la clé privée correspondante. Une licence couvre **un serveur** et tous les modules SmartGUARD de ce serveur.

## Mise en place (faite le 2 octobre 2026)

- Paire de clés créée avec `go run ./cmd/sgkeys keygen -out cles-editeur\smartguard-editeur.sgpriv`.
- Clé privée : `cles-editeur\smartguard-editeur.sgpriv` sur le poste de l'éditeur. Le dossier `cles-editeur/` est ignoré par git : **jamais dans le dépôt**.
- Clé publique copiée dans `internal/license/public.key` : les modules construits depuis la v1.9.0 n'acceptent que les licences signées.

**À faire par l'éditeur :** sauvegarder la clé privée hors ligne, en deux exemplaires (clé USB et coffre ou équivalent). **Perdue, elle ne se remplace pas** : les modules déjà livrés n'accepteraient plus aucune nouvelle licence. **Divulguée, elle permet de forger des licences** : il faudrait créer une nouvelle paire et livrer une nouvelle version du module.

## Émettre une licence

```
go run ./cmd/sgkeys issue -key cles-editeur\smartguard-editeur.sgpriv -customer "Fournisseur X - serveur Y"
```

La commande affiche la clé `SGL1.…` à transmettre au fournisseur (ou à enregistrer dans un fichier texte, que l'assistant d'installation sait charger). Elle contient un identifiant aléatoire, la date d'émission et le titulaire (60 octets au maximum). Pour réémettre une licence existante avec le même identifiant, ajouter `-id <16 caractères hexadécimaux>`.

## Vérifier une licence

```
go run ./cmd/sgkeys verify -pub internal\license\public.key SGL1.…
```

## Limites

- Sans portail, une même clé reste utilisable sur plusieurs serveurs si elle est activée sur chacun : l'activation en ligne (portail) permettra de le détecter (cahier des charges, section 11).
- Il n'y a pas encore de révocation d'une clé déjà émise (prévue avec le portail).
- La clé publique est dans le binaire : quelqu'un qui modifie le programme peut la remplacer. La signature authentifie l'éditeur, elle n'est pas une protection anti-piratage.
