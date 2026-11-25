# Validation wikai

*wikai · cœur wiki · 25 nov. 2026*

30 affirmations remplacent la relecture de la spec. Répondez ligne par ligne ; vos réponses sont enregistrées au fil de l'eau et je les relis depuis la session. La spec sera alignée sur cette fiche, jamais l'inverse.

Réponse vide = ligne non traitée. La fiche gagne sur le document qu'elle valide.

## A — Vos décisions

Je vérifie seulement que ma transcription est fidèle à ce que vous avez dit.

- **A1.** Le CLI s'appelle wikai. C'est un outil de gestion de LLM wiki ; les commandes de home repo y sont un groupe secondaire. *(Intent)*
  - **Réponse : Fidèle**
- **A2.** Le périmètre d'ensemble est wiki + home repo. La capture (yt-dlp, transcription) reste en dehors. *(Non-goals)*
  - **Réponse : Fidèle**
- **A3.** wikai est écrit en Go et s'installe avec Homebrew. *(Constraints)*
  - **Réponse : Fidèle**
- **A4.** wikai remplace tous les scripts déterministes custom de Brain, d'am-home et de dev-home. *(Goals)*
  - **Réponse : Fidèle**
- **A5.** Le binaire fournit des briques ; un fichier de config dans la base dit lesquelles sont actives et avec quels paramètres. Pas de profils codés en Go. *(Approaches)*
  - Conséquence : Un nouveau wiki se monte sans code, tant qu'il combine des briques existantes.
  - **Réponse : Fidèle**
- **A6.** La config a sept blocs : emplacement et style de lien, types de page, index, journal, chemins gelés, provenance, langue. *(wikai.yaml)*
  - **Réponse : Fidèle**
- **A7.** Les règles de frontmatter s'écrivent sous une clé « frontmatter », une entrée par clé, et non comme une liste de clés obligatoires. *(wikai.yaml)*
  - **Réponse : Fidèle**
- **A8.** Quatre projets, dans cet ordre : cœur wiki, commandes home repo, migration (dev-home, puis am-home, puis Brain), marketplace de skills pour Claude Code et Codex. *(The four projects)*
  - **Réponse : Fidèle**

## B — Mes choix par défaut

Les vrais points à trancher. Chaque ligne donne le choix retenu, puis l'alternative.

### Périmètre

- **B1.** Cette spec ne couvre que le projet 1 (cœur wiki). Chacun des trois autres projets aura sa propre spec. *(spec)*
  - Sinon : Une seule spec pour les quatre projets.
  - **Réponse : Choix retenu**
- **B2.** Aucune page existante n'est modifiée pour s'adapter à wikai : c'est la config qui s'adapte aux bases. *(spec)* — _Jamais discuté_
  - Sinon : Faire converger les trois bases vers un format commun.
  - **Réponse : Choix retenu**
- **B3.** Une règle qu'aucune brique ne couvre reste un contrôle du projet, en dehors de wikai. *(spec)* — _Jamais discuté_
  - Conséquence : Exemple : les gardes de test_home.py propres à am-home restent un test d'am-home tant que le projet 2 ne les couvre pas.
  - Sinon : La config peut déclarer des contrôles custom (un script par contrôle).
  - **Réponse : Choix retenu**
- **B4.** wikai n'appelle ni modèle ni réseau, et ne fait pas de recherche dans la base. *(spec)* — _Jamais discuté_
  - **Réponse : D'accord**
- **B5.** « wikai init » crée la config, les dossiers, les squelettes et l'index. Il n'écrit ni AGENTS.md ni procédures : ils viennent avec les skills du projet 4. *(spec)* — _Jamais discuté_
  - Conséquence : Entre le projet 1 et le projet 4, monter un nouveau wiki demande encore d'écrire son contrat à la main.
  - Sinon : « init » écrit aussi un contrat d'agent minimal.
  - **Réponse : Choix retenu**
### Config

- **B6.** Une clé de frontmatter se décrit avec : obligatoire, format (date, liste, texte), valeurs permises, non vide, vide autorisé. Rien d'autre. *(spec)*
  - Sinon : Ajouter des règles plus riches (expression régulière par clé).
  - **Réponse : Choix retenu**
- **B7.** Une clé obligatoire seulement pour certaines pages (entity_kind quand kind vaut entity) se déclare dans la variante de page concernée, sans syntaxe de condition. *(spec)* — _Jamais discuté_
  - Sinon : Abandonner cette règle et la laisser au lint fait par le modèle.
  - **Réponse : Choix retenu**
- **B8.** Une clé inconnue dans wikai.yaml est une erreur. *(spec)* — _Jamais discuté_
  - Sinon : Un simple avertissement.
  - **Réponse : Choix retenu**
- **B9.** Le format des entrées de journal est réglable (libellés, séparateur, une ligne ou plusieurs, pages en liens), pour continuer log.md de Brain à l'identique. *(spec)* — _Jamais discuté_
  - Sinon : Un format unique : les nouvelles entrées de Brain changent de forme à partir de la migration.
  - **Réponse : Choix retenu**
- **B10.** La langue (en ou fr) ne règle que les textes que wikai écrit dans les pages. Les messages du CLI sont en anglais, pour Brain aussi. *(spec)* — _Jamais discuté_
  - Sinon : Messages du CLI traduits selon la langue de la base.
  - **Réponse : Choix retenu**
### Comportement

- **B11.** Le frontmatter est lu par un vrai parseur YAML. Une page que les anciens scripts toléraient mais qui n'est pas du YAML valide devient un constat. *(spec)* — _Jamais discuté_
  - Conséquence : Le passage d'acceptation montre ces pages s'il y en a ; le projet 3 décide quoi en faire.
  - Sinon : Reproduire la tolérance des deux parseurs maison.
  - **Réponse : Choix retenu**
- **B12.** Une seule règle de gel pour toutes les bases : comparaison avec la branche distante. *(spec)* — _Jamais discuté_
  - Conséquence : Pour Brain, c'est plus strict : une source brute modifiée dans un commit local non poussé est aussi signalée.
  - Sinon : Brain garde sa règle actuelle (comparaison avec le dernier commit local).
  - **Réponse : Choix retenu**
- **B13.** « wikai move » accepte un archivage depuis n'importe quel type de page. *(spec)* — _Jamais discuté_
  - Sinon : Archivage limité aux designs, comme new.py aujourd'hui.
  - **Réponse : Choix retenu**
- **B14.** Les squelettes sont réécrits au projet 3 avec {{title}}, {{date}}, {{slug}}, {{summary}}. wikai ne lit pas les anciennes formes. *(spec)* — _Jamais discuté_
  - Sinon : wikai comprend aussi les squelettes actuels.
  - **Réponse : Choix retenu**
- **B15.** Au projet 1, une page de repo se crée avec un titre explicite et « --set origin=… ». La commande qui déduit le nom depuis l'origin git arrive au projet 2. *(spec)* — _Jamais discuté_
  - Sinon : La reprendre dès le projet 1.
  - **Réponse : Choix retenu**
- **B16.** Le contrôle d'index reste désactivé pour Brain, qui n'en a pas aujourd'hui. Son activation se décide au projet 3. *(spec)* — _Jamais discuté_
  - Conséquence : « wikai new » et « wikai move » tiennent quand même l'index de Brain à jour.
  - Sinon : L'activer dès le projet 1 et traiter les constats.
  - **Réponse : Choix retenu**
- **B17.** La vérification du format des citations de code se règle par base. dev-home la garde désactivée ; le projet 3 décide. *(spec)* — _Jamais discuté_
  - Sinon : L'activer pour dev-home dès maintenant.
  - **Réponse : Choix retenu**
### Gardes

- **B18.** Quand un chemin relève de plusieurs règles : ajout seul l'emporte, puis gelé, puis protégé. *(spec)* — _Jamais discuté_
  - Conséquence : Dans Brain, l'agent peut ajouter une ligne au manifeste et un rapport dans _meta/lint/, et rien d'autre sous _meta. Il n'écrit jamais dans 10-raw/zotero.
  - **Réponse : D'accord**
- **B19.** Le hook de garde vise Claude Code seulement. Un hook pour Codex n'est pas prévu au projet 1. *(spec)* — _Jamais discuté_
  - Sinon : Prévoir Codex dès le projet 1.
  - **Réponse : Choix retenu**
### Livraison

- **B20.** Même recette que den : goreleaser et cask Homebrew dans PillowPillow/tap, pour macOS et Linux. *(spec)* — _Jamais discuté_
  - **Réponse : À discuter** — oui, attention d'utiliser toutes les libs/frameworks/langage en derniere version sans vuln. on vise le no vuln
- **B21.** Les fixtures reprises d'am-home sont réécrites avec des noms fictifs : aucun nom réel de client, de personne, de dépôt ou d'hôte dans le dépôt wikai. *(spec)* — _Jamais discuté_
  - **Réponse : D'accord**
- **B22.** Critère d'acceptation : avec une config écrite pour chaque base et gardée hors des bases, « wikai check » ne signale aucun constat sur Brain, am-home et dev-home, sans les modifier. *(spec)* — _Jamais discuté_
  - **Réponse : D'accord**

## C — Ce que vous n'avez pas à valider

Détail technique porté par la spec, le plan et les tests : structure des paquets Go, noms des contrôles, codes de sortie (0, 1, 2), format exact des messages, portage des 29 fixtures d'am-home, calcul du hash du manifeste, règles de résolution des wikilinks et des ancres. Si un de ces points contredit A ou B, c'est A ou B qui gagne.Faits vérifiés pendant l'analyse : les trois contrôles actuels passent (Brain 129 notes, am-home 148 pages, dev-home 15 pages) ; `den` est distribué par goreleaser et un cask Homebrew dans `PillowPillow/tap`.De mémoire, non vérifié : le format d'une marketplace Codex (à vérifier au projet 4).

## Exceptions

| Ligne | Réponse | Ce qui est retenu |
|---|---|---|
| B20 | À discuter : « oui, attention d'utiliser toutes les libs/frameworks/langage en derniere version sans vuln. on vise le no vuln » | La recette de den est retenue (goreleaser, cask Homebrew dans `PillowPillow/tap`, macOS et Linux). S'y ajoute une exigence : Go et chaque dépendance dans leur dernière version stable, vérifiée à la source au moment du plan et non de mémoire ; zéro vulnérabilité connue ; `govulncheck` tourne en CI et bloque le merge et la release au premier constat. |
