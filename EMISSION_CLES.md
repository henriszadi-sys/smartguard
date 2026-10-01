# Émission des licences (fournisseur)

Ce document est destiné au fournisseur du logiciel. Il ne doit pas être livré aux clients.

## Principe

Les licences sont signées avec Ed25519. Le fournisseur garde la **clé privée** ; le module installé chez le client ne contient que la **clé publique** et refuse toute clé qui n'a pas été émise avec la clé privée correspondante.

## Mise en place (une seule fois)

1. Créer la paire de clés, **hors du dépôt** (le fichier `*.sgpriv` est ignoré par git) :
   ```
   go run ./cmd/sgkeys keygen -out C:\Secrets\fournisseur.sgpriv
   ```
   La commande refuse d'écraser un fichier existant et affiche la clé publique.
2. Sauvegarder `fournisseur.sgpriv` dans un endroit sûr. **Perdue, elle ne se remplace pas** : toutes les licences émises devraient être réémises avec une nouvelle paire. **Divulguée, elle permet de forger des licences.**
3. Copier la clé publique affichée dans `internal/license/public.key` (une seule ligne).
4. Reconstruire les exécutables (commandes de `CLAUDE.md`) et régénérer l'archive. Un module construit avec `public.key` renseignée n'accepte plus que les clés signées.

## Émettre une licence

```
go run ./cmd/sgkeys issue -key C:\Secrets\fournisseur.sgpriv -customer "Société X"
```

La commande affiche la clé `SGL1.…` à transmettre au client. Elle contient un identifiant aléatoire, la date d'émission et le titulaire (60 octets au maximum). Pour réémettre une licence existante avec le même identifiant, ajouter `-id <16 caractères hexadécimaux>`.

## Vérifier une licence

```
go run ./cmd/sgkeys verify -pub <clé publique ou fichier> SGL1.…
```

## Limites

- Une même clé reste utilisable sur plusieurs serveurs si elle est activée sur chacun : il n'y a pas de serveur central pour le détecter (voir le cahier des charges, section 4).
- Il n'y a pas de révocation d'une clé déjà émise.
- La clé publique est dans le binaire : quelqu'un qui modifie le programme peut la remplacer. La signature authentifie l'émetteur, elle n'est pas une protection anti-piratage.
