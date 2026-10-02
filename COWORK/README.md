# COWORK — documents produits avec Claude

Ce dossier rassemble les documents de pilotage et de recette produits pendant le développement de SmartGUARD avec Claude (Cowork). Il est versionné sur GitHub avec le code.

**Ne jamais y placer** : exécutable, archive (`.zip`), clé privée (`*.sgpriv`), licence émise, mot de passe ou donnée d'un client réel.

Les documents de référence restent à la racine du dépôt (`Cahier_des_charges_SmartGUARD.md` et `.pdf`, `CLAUDE.md`, `LISEZMOI.md`, `EMISSION_CLES.md`) et dans `docs/` (plan de développement, état des lieux). COWORK contient les **exports et productions** : PDF, tableurs, procès-verbaux, comptes rendus.

## Index

| Dossier | Fichier | Contenu | Version / date | Statut |
|---|---|---|---|---|
| recette | `PV_recette_jalon_A_v1.10.0.md` (+ `.pdf`) | Procès-verbal de recette du jalon A : critères, preuves déjà obtenues, scénarios S1 à S12 sur machines réelles | Module v1.10.0, cahier v1.20 — 2 oct. 2026 | À dérouler sur machines réelles |
| plans | `Plan_de_developpement_SmartGUARD_v1.6.pdf` | Plan de développement en PDF (source : `docs/Plan_de_developpement_SmartGUARD.md`) | v1.6 — 2 oct. 2026 | À jour |
| tarification | `SmartGUARD_Tarification.xlsx` | Grille tarifaire, simulateur et exemples (formules modifiables) | Cahier v1.17 (prix inchangés jusqu'à v1.20) — 1er oct. 2026 | Prix des options : proposition |
| tarification | `SmartGUARD_Tarification.pdf` | Grille tarifaire en PDF | Cahier v1.17 — 1er oct. 2026 | Prix des options : proposition |
| outils | `md2pdf.py` | Conversion Markdown → PDF des documents (option `paysage` pour les grands tableaux) | — | — |

## Règles de rangement

- Un sous-dossier par nature : `recette/`, `plans/`, `tarification/`, `comptes-rendus/`, `cahier-des-charges/` (exports datés), `outils/`.
- Nom de fichier : sujet, puis version ou date (`AAAA-MM-JJ`) : `PV_recette_jalon_A_v1.10.0.pdf`, `CR_reunion_2026-10-15.md`.
- Un document remplacé n'est pas supprimé : il est déplacé dans `archives/` du même sous-dossier.
- Chaque ajout ou mise à jour est inscrit dans l'index ci-dessus, dans le même commit.
