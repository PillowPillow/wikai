# Validation plan wikai

*wikai · plan d'implémentation du cœur wiki · 25 nov. 2026*

29 affirmations remplacent la relecture du plan de 700 lignes. Répondez ligne par ligne ; vos réponses sont enregistrées au fil de l'eau et je les relis depuis la session. Le plan sera aligné sur cette fiche, jamais l'inverse.

## Contexte

La spec du cœur de wikai est approuvée. Le plan la découpe en 17 tâches, chacune avec ses tests écrits avant le code : la config, les neuf contrôles de « wikai check », les commandes d'écriture, le manifeste, les gardes, « wikai init », la distribution Homebrew, puis le passage d'acceptation : « wikai check » lancé sur Brain, am-home et dev-home, sans les modifier. En écrivant le plan, j'ai dû trancher des points que la spec laisse ouverts ou sur lesquels elle se contredit. Un « oui » ici fixe ce que le code fera ; un « non » corrige le plan avant la première ligne de code.

## Glossaire

- **base** : Un wiki géré par wikai : le vault Brain, ou le dossier knowledge/ d'am-home ou de dev-home.
- **frontière** : Le dossier qui contient wikai.yaml : la racine du dépôt git pour un home, la racine du vault pour Brain. Aucun lien ne doit en sortir.
- **racine des pages** : La clé « root » de wikai.yaml : le dossier, sous la frontière, où vivent les pages. « knowledge » pour un home, « . » pour Brain.
- **chemin gelé** : Dossier déclaré sous « frozen » : on peut y ajouter un fichier, jamais modifier, renommer ou supprimer un fichier existant (10-raw dans Brain, archive et log dans un home).
- **fichier en ajout seul** : Fichier déclaré sous « append_only » : il ne peut que grandir par la fin (le manifeste de Brain).
- **chemin protégé** : Chemin déclaré sous « protected » : l'agent n'y écrit jamais, l'humain librement.
- **fichier égaré** : Fichier markdown posé sous la racine des pages mais dans aucun dossier de type de page ni chemin gelé (contrôle « stray »).
- **squelette** : Fichier modèle que « wikai new page » copie et remplit pour créer une page.
- **preset** : Jeu de fichiers de départ écrit par « wikai init » : un wikai.yaml lisible, les dossiers, les squelettes, l'index.
- **jeu de test repris** : Une des 29 petites bases d'exemple d'am-home, chacune avec la liste exacte des constats attendus ; wikai doit rendre la même liste.
- **style de lien** : La clé « links » de wikai.yaml : « markdown » pour les deux homes (liens relatifs [texte](chemin.md)), « wikilink » pour Brain ([[Nom]]).
- **hook de garde** : La commande « wikai guard hook », branchée comme hook PreToolUse de Claude Code : elle reçoit chaque écriture de fichier que l'agent s'apprête à faire et peut la refuser.

Réponse vide = ligne non traitée. La fiche gagne sur le document qu'elle valide.

## A — Vos décisions

Je vérifie seulement que ma transcription est fidèle à ce que vous avez dit.

- **A1.** Le plan ne couvre que le projet 1 des quatre prévus : le cœur wiki. Le projet 2 (commandes de home repo), le projet 3 (migration de dev-home, am-home puis Brain) et le projet 4 (marketplace de skills) n'y figurent pas. *(votre demande du 25 nov.)*
  - **Réponse : Fidèle**
- **A2.** Go et chaque dépendance sont dans leur dernière version stable, vérifiée à la source. govulncheck tourne en CI à chaque push et avant chaque release ; la moindre vulnérabilité signalée bloque. *(votre réponse sur la livraison, fiche de validation de la spec)*
  - **Réponse : Fidèle**

## B — Mes choix par défaut

Les vrais points à trancher. Chaque ligne donne le choix retenu, puis l'alternative.

### Dépendances

- **B1.** La bibliothèque YAML est go.yaml.in/yaml/v4 en 4.0.0, la dernière version majeure du fork maintenu, et non gopkg.in/yaml.v3 que la spec nomme. *(plan · contraintes globales)* — _S'écarte de la spec_
  - **Ce que montre la figure** : Les deux forks sont maintenus ; la v4 est la seule qui évolue encore, la v3 du fork ne reçoit que des correctifs de sécurité.

    | Critère | go.yaml.in/yaml/v3 | gopkg.in/yaml.v3 | go.yaml.in/yaml/v4 |
    |---|---|---|---|
    | Dernière version | 3.0.6, oct. 2026 | 3.0.1, mai 2022 | 4.0.0, nov. 2026 |
    | Maintenue | oui, correctifs de sécurité seulement | non, dépôt archivé | oui, développement actif |
    | API | identique à celle de la spec | celle de la spec | lecture compatible avec la v3 |

  - Conséquence : Vérifié dans sa documentation : la v4 garde les fonctions de lecture de la v3 (Unmarshal, Decoder, refus des clés inconnues) ; seule l'écriture des lecteurs sur mesure change, et le plan n'en a qu'un. La v3 du fork ne reçoit plus que des correctifs de sécurité.
  - Option Fork maintenu, v3 : go.yaml.in/yaml/v3 en 3.0.6 : même API que la bibliothèque de la spec, correctifs de sécurité seulement.
  - Option Celle de la spec : gopkg.in/yaml.v3 en 3.0.1 : aucune vulnérabilité ouverte connue, mais dépôt archivé, donc plus aucun correctif à venir.
  - Option Fork maintenu, v4 (choix retenu) : go.yaml.in/yaml/v4 en 4.0.0 : la dernière version majeure, sortie le 4 nov. 2026 ; lecture compatible avec la v3.
  - Réponse à la note : Vous avez choisi la v4 : le plan passe à go.yaml.in/yaml/v4 en 4.0.0.
  - **Réponse : Fork maintenu, v4**
- **B2.** wikai a trois dépendances directes et pas une de plus : cobra, la bibliothèque YAML, et golang.org/x/text pour comparer les noms accentués. *(plan · contraintes globales)* — _Jamais discuté_
  - Sinon : Autoriser d'autres dépendances quand elles évitent d'écrire du code.
  - **Réponse : Choix retenu**

### Config

- **B3.** Dans wikai.yaml, « root » et « protected » se lisent depuis la frontière ; tous les autres chemins se lisent depuis la racine des pages. *(plan · contraintes globales)* — _La spec se contredit_
  - **Ce que montre la figure** : Avec le choix retenu, un home écrit « glossary » ; avec l'alternative, il écrit « knowledge/glossary ».

    | Critère | Depuis la racine des pages | Tout depuis la frontière |
    |---|---|---|
    | Dossier d'un type de page | folders: [glossary] | folders: [knowledge/glossary] |
    | Chemin gelé | path: archive | path: knowledge/archive |
    | Chemin protégé | wikai.yaml (depuis la frontière) | wikai.yaml |
    | Config de Brain (racine des pages = frontière) | inchangée | inchangée |

  - Conséquence : L'exemple de config de la spec pour un home reste valable tel quel. La phrase de la spec « les chemins de la config sont relatifs à la frontière » est corrigée.
  - Sinon : Tous les chemins se lisent depuis la frontière : la config d'un home répète « knowledge/ » devant chaque dossier.
  - **Réponse : Choix retenu**
- **B4.** L'option « --dir » ne s'emploie qu'avec « --config » ; seule, elle est refusée avec un message et le code de sortie 2. *(plan · tâche 2)* — _Jamais discuté_
  - Sinon : Seule, « --dir » désigne le dossier d'où part la recherche de wikai.yaml.
  - **Réponse : Choix retenu**
- **B27.** wikai.yaml porte une clé « version », obligatoire, qui vaut 1 aujourd'hui. wikai refuse un fichier sans cette clé ou d'une version qu'il ne connaît pas, et le message dit quelle version il attend. *(plan · tâche 2)* — _Ajouté à votre demande_
  - Conséquence : Un changement incompatible du format passera en version 2 ; la commande qui convertit un fichier de version 1 sera écrite à ce moment-là, pas au projet 1.
  - Sinon : Écrire dès maintenant une commande « wikai config migrate », qui ne fait rien tant qu'il n'existe qu'une version.
  - Réponse à la note : Vous demandiez une version de la config pour gérer les futures mises à jour et migrations : la spec la prévoyait déjà, cette ligne fixe ce que wikai en fait.
  - **Réponse : Choix retenu**

### Contrôles

- **B5.** Quand le contrôle des fichiers égarés est actif (« stray: true », le défaut, celui des homes), un fichier égaré reçoit ce seul constat et ne compte pas comme page. *(plan · tâche 4)* — _S'écarte de la spec_
  - Conséquence : C'est ce que font les scripts actuels des homes : leur jeu de test repris attend un constat par fichier égaré et aucun autre. La spec disait que tout fichier markdown est une page.
  - Sinon : Un fichier égaré est aussi une page : il reçoit en plus les constats de frontmatter, de liens et d'orphelin.
  - **Réponse : Choix retenu**
- **B6.** Quand l'index n'accepte que des liens vers des pages (« other_links: forbid », le défaut, celui des homes), c'est le contrôle d'index qui signale ses liens morts ; le contrôle des liens ne lit pas l'index. *(plan · tâche 6)* — _S'écarte de la spec_
  - Conséquence : Une entrée d'index morte donne un constat, pas deux. Pour Brain, dont l'index est une page avec frontmatter, le contrôle des liens lit l'index normalement.
  - Sinon : Les deux contrôles lisent l'index : une entrée morte donne deux constats.
  - **Réponse : Choix retenu**
- **B7.** Une page dont le type n'est pas déclaré dans wikai.yaml, ou n'est pas celui de son dossier, reçoit ce constat et seulement les règles de frontmatter écrites pour toutes les pages ; les règles propres à un type ne lui sont pas appliquées. *(plan · tâche 5)* — _Jamais discuté_
  - Sinon : Elle reçoit en plus les règles du type qu'elle déclare : une page « repo » rangée dans glossary/ est aussi signalée pour sa clé origin manquante.
  - **Réponse : Choix retenu**
- **B8.** En style de lien wikilink, la cible d'un lien est cherchée parmi tous les fichiers présents sur le disque, suivis par git ou non ; en style de lien markdown, parmi les seuls fichiers suivis par git. *(plan · tâche 4)* — _Jamais discuté_
  - Conséquence : Dans Brain, une note tout juste créée et pas encore ajoutée à git est une cible valide, comme avec le script actuel.
  - Sinon : Fichiers suivis par git dans les deux styles : dans Brain, un lien vers une note pas encore ajoutée à git est un lien cassé.
  - **Réponse : Choix retenu**
- **B9.** En style de lien wikilink, l'ancre de titre (« [[Note#Titre]] ») est vérifiée comme l'ancre de bloc (« [[Note#^bloc]] ») : la note cible doit contenir un titre de ce texte, comparé sans tenir compte de la casse. *(plan · tâche 6)* — _Jamais discuté_
  - Conséquence : Le script actuel de Brain ne vérifie pas ces ancres : le passage d'acceptation peut faire apparaître des liens de Brain dont le titre visé n'existe plus.
  - Sinon : Seule l'ancre de bloc est vérifiée, comme dans le script actuel de Brain.
  - Réponse à la note : Vous avez choisi de vérifier aussi les ancres de titre : c'est maintenant le choix retenu.
  - **Réponse : Choix retenu**
- **B10.** « wikai check » signale aussi, sous le contrôle « frozen » et par comparaison avec la branche distante, un fichier en ajout seul dont des lignes existantes ont changé. *(plan · tâche 9)* — _Jamais discuté_
  - Conséquence : La garde de commit le signale déjà au moment du commit ; « wikai check » le rattrape si la garde n'est pas installée.
  - Sinon : Seule la garde de commit surveille les fichiers en ajout seul ; « wikai check » les ignore.
  - **Réponse : Choix retenu**

### Écriture

- **B11.** « --set clé=valeur » de « wikai new page » ne fait que remplir « {{clé}} » dans le squelette ; il n'ajoute pas de clé au frontmatter. *(plan · tâche 10)* — _Jamais discuté_
  - Conséquence : Une page de repo reçoit son origin parce que son squelette contient « origin: {{origin}} ».
  - Sinon : « --set » écrit aussi la clé dans le frontmatter quand le squelette ne la prévoit pas.
  - **Réponse : Choix retenu**
- **B12.** Une page créée dans un dossier absent de la liste « include » de l'index est créée sans ligne d'index. *(plan · tâche 10)* — _Jamais discuté_
  - Conséquence : Dans Brain, seules les pages de 20-wiki reçoivent leur ligne ; une page de 30-notes ou de 40-veille n'en reçoit pas.
  - Sinon : « wikai new page » refuse de créer une page que l'index ne couvre pas.
  - **Réponse : Choix retenu**
- **B13.** Dans une entrée de log.md de Brain écrite par « wikai new log », les pages créées tiennent sur une ligne « créé », séparées par des virgules, et les pages modifiées sur une ligne « modifié » ; chaque « --note » donne sa propre ligne « note ». *(plan · tâche 11)* — _Jamais discuté_
  - Sinon : Les « --note » sont elles aussi jointes sur une seule ligne « note ».
  - **Réponse : Choix retenu**
- **B14.** En style de lien markdown, quand « wikai move » change une page de dossier, il recalcule aussi les liens relatifs écrits dans la page déplacée. *(plan · tâche 12)* — _Jamais discuté_
  - Conséquence : Une page archivée garde des liens qui pointent vers les mêmes fichiers. Le script actuel ne le fait pas.
  - Sinon : Comme aujourd'hui : seuls les liens vers la page déplacée sont réécrits ; ses propres liens peuvent casser.
  - **Réponse : Choix retenu**

### Manifeste

- **B15.** Une ligne écrite par « wikai manifest append » prend la forme compacte par défaut de Go : mêmes clés dans le même ordre que manifest.py, sans espace après les virgules et les deux-points. *(plan · tâche 13)* — _Jamais discuté_
  - Conséquence : Le manifeste de Brain mélange deux formes à partir de la première ligne écrite par wikai ; les anciennes lignes restent lisibles et ne sont pas réécrites.
  - Sinon : La ligne a exactement la forme de celles de manifest.py, à l'octet près, espaces compris.
  - Réponse à la note : Vous avez choisi la forme compacte de Go : c'est maintenant le choix retenu.
  - **Réponse : Choix retenu**

### Gardes

- **B16.** Le hook de garde refuse l'écriture quand le wikai.yaml de la base est invalide ou que l'événement envoyé par Claude Code est illisible. *(plan · tâche 14)* — _Jamais discuté_
  - Conséquence : Une faute de frappe dans wikai.yaml bloque toutes les écritures de l'agent dans la base jusqu'à sa correction.
  - Sinon : Le hook laisse passer quand il ne peut pas décider : une config cassée ne bloque jamais l'agent, mais ne protège plus rien.
  - **Réponse : Choix retenu**
- **B26.** Le hook de garde applique les règles de chaque base au-dessus du fichier visé (chemin résolu et chemin écrit) et de la base du projet Claude Code ; il refuse dès qu'une base refuse. Il laisse passer l'écriture quand aucune base ne contient le fichier. *(plan · tâche 14 ; révisé le 2026-11-26)*
  - Conséquence : Le même hook peut être installé au niveau global : il garde aussi une base voisine ou imbriquée, et un wikai.yaml imbriqué ne masque pas les règles de la base qui le contient.
  - Sinon : Le hook ne garde que la base au-dessus du dossier du projet ; une écriture dans une autre base passe.
  - **Réponse : Choix révisé par l'humain le 2026-11-26 (toutes les bases sur le chemin)**
- **B17.** Le dossier .git est toujours un chemin protégé, sans qu'il faille l'écrire dans wikai.yaml. *(plan · tâche 14)* — _Jamais discuté_
  - Conséquence : L'agent ne peut pas écrire dans .git avec ses outils d'édition, même si la config l'oublie.
  - Sinon : Le dossier .git n'est protégé que si wikai.yaml le liste sous « protected ».
  - **Réponse : Choix retenu**

### Init

- **B18.** Le preset pour un vault Obsidian est générique et en anglais : dossiers wiki/, raw/ et notes/, et trois sortes de page wiki (concept, summary, entity). Il ne reprend pas l'arborescence numérotée de Brain. *(plan · tâche 15)* — _Jamais discuté_
  - Sinon : Le preset est une copie de la structure de Brain (00-inbox, 10-raw, 20-wiki…, textes en français).
  - **Réponse : Choix retenu**

### Livraison

- **B19.** wikai s'installe par le cask Homebrew et aussi par un script install.sh écrit sur le modèle de celui de den, vérifié en CI. *(plan · tâche 16)* — _Jamais discuté_
  - Conséquence : La CI télécharge la dernière release pour essayer le script : ce contrôle ne peut tourner qu'après la première release.
  - Sinon : Le cask Homebrew seulement, sans install.sh.
  - Réponse à la note : Vous avez choisi de reprendre aussi install.sh : c'est maintenant le choix retenu.
  - **Réponse : Choix retenu**
- **B28.** Le dépôt wikai porte la licence GPL-3.0, celle de votre dépôt tea-shot. *(plan · tâche 16)*
  - Conséquence : Vérifié sur GitHub : tea-shot est sous GNU General Public License v3.0. Quiconque redistribue un wikai modifié doit le publier sous la même licence. den reste sous Apache-2.0.
  - Sinon : Apache-2.0, comme den.
  - Réponse à la note : Vous demandiez la GPL comme tea-shot : cette ligne remplace celle qui proposait Apache-2.0.
  - **Réponse : Choix retenu**
- **B21.** La première release porte le numéro v0.1.0. *(plan · tâche 16)* — _Jamais discuté_
  - Conséquence : C'est vous qui poussez le tag, une fois le passage d'acceptation réussi. Pousser le tag publie la release et le cask ; cela ne s'annule pas.
  - Sinon : La première release est v1.0.0.
  - **Réponse : Choix retenu**

### Acceptation

- **B22.** Les trois configs d'acceptation (une par base : Brain, am-home, dev-home) restent sur votre machine, hors du dépôt wikai ; le dépôt ne garde que le script qui les lance. *(plan · tâche 17)* — _Jamais discuté_
  - Conséquence : Le script lit le dossier des configs dans une variable d'environnement. Aucun nom de dossier de vos bases n'entre dans le dépôt ; les configs ne sont ni sauvegardées ni relues avec le code.
  - Sinon : Les trois configs sont versionnées dans le dépôt wikai, sous acceptance/.
  - Réponse à la note : Vous avez choisi de garder les configs hors du dépôt : c'est maintenant le choix retenu.
  - **Réponse : Choix retenu**
- **B23.** Le passage d'acceptation exige zéro constat sur les trois bases, sans exception : la release attend. *(plan · tâche 17)* — _Jamais discuté_
  - Conséquence : Si une page a un frontmatter que le vrai parseur YAML refuse, c'est vous qui la corrigez dans la base avant la release : ce projet n'écrit jamais dans une base. Une source brute modifiée dans un commit local se règle en poussant ce commit. Tout autre constat est reproduit en jeu de test, puis corrigé dans wikai ou dans la config.
  - Sinon : Deux sortes de constat ne bloquent pas la release, parce que la spec les annonce comme écarts connus : un frontmatter que le vrai parseur YAML refuse, et une source brute de Brain modifiée dans un commit local pas encore poussé.
  - Réponse à la note : Vous avez choisi zéro constat strict : c'est maintenant le choix retenu. La conséquence dit qui corrige quoi.
  - **Réponse : Choix retenu**

### Exécution

- **B24.** Un seul plan pour les 17 tâches, exécuté d'une traite sur la branche actuelle spec/wikai-core, avec une seule fusion à la fin. *(plan · en-tête)* — _Jamais discuté_
  - Sinon : Trois plans livrés l'un après l'autre : « wikai check » ; puis écriture, manifeste et gardes ; puis init et distribution.
  - **Réponse : Choix retenu**
- **B25.** Chaque tâche est codée par un sous-agent démarré sans le contexte de cette session, puis relue par un second sous-agent tout aussi neuf, avant la tâche suivante. *(plan · passage à l'exécution)* — _À choisir_
  - Conséquence : C'est le mode le plus sûr et le plus coûteux : un contexte neuf par tâche et par relecture.
  - Sinon : Je code les 17 tâches dans une seule session, avec une seule relecture de toute la branche à la fin : plus rapide et moins cher.
  - **Réponse : Choix retenu**

## C — Ce que vous n'avez pas à valider

Détail porté par le plan et ses tests : noms et signatures des fonctions Go, découpage en fichiers, texte exact des constats, liste des 49 jeux de test (29 repris d'am-home, 20 nouveaux), les expressions de lecture des liens et des citations, les codes de sortie, les fichiers de CI. Si un de ces points contredit A ou B, c'est A ou B qui gagne. Faits vérifiés à la source le 25 nov. 2026 : Dernières versions stables : Go 1.27.4 (installé sur votre machine), cobra 1.10.2, golang.org/x/text 0.44.0, govulncheck 1.10.0, goreleaser 2.20.3. Le dépôt d'origine de gopkg.in/yaml.v3 est archivé depuis avril 2025 ; sa dernière version date de 2022. La base de vulnérabilités Go n'y signale rien d'ouvert. Les scripts de dev-home ne diffèrent de ceux d'am-home que par les trois écarts déjà nommés dans la spec (pas de citations de code, origin/master, textes français). Les six empreintes de référence du test du manifeste sortent de manifest.py, lancé sur des notes synthétiques. Dans le jeu de test repris « type erroné », la page est déclarée « repo » dans glossary/ et n'a pas de clé origin ; dans « entrée d'index morte », l'index pointe vers un fichier absent. Préalables à la release, hors plan : le dépôt GitHub PillowPillow/wikai existe, et il porte le même jeton d'accès au dépôt du tap Homebrew que den.
