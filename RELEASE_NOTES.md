# `stcopenmpp` 0.3 Release Notes/Notes de Mise à Jour

<details open>

<summary>English</summary>

### Upstream sync

Selectively merged upstream changes from [OpenM++ v1.17.14](https://github.com/openmpp/main/releases/tag/v1.17.14) (2026-09-01) 
and [OpenM++ v1.17.15](https://github.com/openmpp/main/releases/tag/v1.17.15) (2026-10-01).

### Docker changes

See [ompp/ompp-docker/README.md](ompp/ompp-docker/README.md) for updates to `stcopenmpp` 
build dependencies.

### Core changes

- **Output tables:** `null` values have been replaced with `""` values.
- **Microdata table:** the primary key on `run_id` and `entity_key` has been removed from the
  microdata data table in the model database. This change is relevant when model users are
  outputting microdata in database mode. An alert will appear in the UI notifying model users 
  of this change: "The microdata table no longer has a primary key on run_id and entity_key. 
  If duplicate rows are written to the microdata table, the model run will not fail. 
  But duplicate rows may be present and results may be double counted."

### UI changes

- **Caching:** new caching system remembers various UI states as set by the model user. 
  This system can be cleared and turned on/off from `Settings` > `Browser cache`.
- **Modals:** model parameters on the `Input Scenarios` tab now open in modals (pop-up windows) 
  instead of tabs.
- **Checkbox lists:** on the `Run the Model` tab, the two output table lists have been
  replaced with a single checkbox list. Likewise for the microdata entity lists.
- **Microdata output toggle:** under the `Advanced Run Options` accordion on the 
  `Run the Model` tab, a microdata output toggle has been added so that model users
  can decide whether to output microdata to the database or to a text file (.csv format).
 - **Styling:** the color scheme has been updated, new icons have been added and more space
  has been placed between selected elements.
  
### Toy models

A new toy model, `BMIGen`, now ships with `stcopenmpp`. This model is a microsimulation
model of body mass index and mortality. `BMIGen` is a reduced version of Statistic Canada's
[Population Health Model (`POHEM`)](https://www.statcan.gc.ca/en/microsimulation/index#pohem), 
which is a microsimulation model of select chronic diseases and risk factors in which 
the individual is the unit of analysis.

### Bundled plugin updates

- **auditr**: upgraded to v0.33.
- **openmpp:** upgraded to v0.0.2.9000.
- **inst_ubuntu:** initial version added. This plugin provides shell scripts for installing 
  Statistic Canada's OpenM++ distribution (`stcopenmpp`) on Ubuntu, along with the compiler 
  toolchain and build tools it needs.
  
### Documentation
 
A new [Building from source](docs/building-from-source) guide has been added to the `docs` folder with 
instructions on how to build `stcopenmpp` manually from source on both Linux and Windows.

</details>

<details open>

<summary>Français</summary>

### Synchronisation en amont

Fusion sélective des modifications en amont de [OpenM++ v1.17.14](https://github.com/openmpp/main/releases/tag/v1.17.14) (01/09/2026) 
et [OpenM++ v1.17.15](https://github.com/openmpp/main/releases/tag/v1.17.15) (01/10/2026).

### Changements de Docker

Voir [ompp/ompp-docker/README.md](ompp/ompp-docker/README.md) pour les mises à jour des dépendances de compilation de `stcopenmpp`.

### Changements fondamentaux

- **Tables de sortie :** les valeurs `null` ont été remplacées par des valeurs `""`.
- **Tableau de microdonnées :** la clé primaire sur `run_id` et `entity_key` a été supprimée de la section
Tableau de données de microdonnées dans la base de données modèle. Ce changement est pertinent lorsque les utilisateurs du modèle sont
Sortie de microdonnées en mode base de données. Une alerte apparaîtra dans l’interface utilisateur pour informer les utilisateurs du modèle.
de ce changement : « Le tableau de microdonnées n’a plus de clé primaire sur run_id et entity_key. Si des lignes en double sont écrites dans 
la table de microdonnées, l’exécution du modèle n’échouera pas. Mais des lignes en double peuvent être présentes et les résultats peuvent être 
comptés en double. »

### Modifications de l’interface utilisateur

- **Mise en cache :** Le nouveau système de mise en cache se souvient des différents états de l’interface utilisateur tels que définis 
  par l’utilisateur du modèle. Ce système peut être effacé et activé/désactivé à partir de `Règlage` > `Cache du navigateur`.
- **Modaux** : les paramètres du modèle sur l’onglet 'Scénarios d’entrée' sont désormais ouverts dans les modaux (fenêtres contextuelles) 
  au lieu d’onglets.
- **Listes de cases à cocher :** dans l’onglet `Exécutez le modèle`, les deux listes de tableaux de résultats ont été
  remplacé par une seule liste de cases à cocher. Cela vaut égalment pour les listes d’entités de microdonnées.
- **Basculement de sortie de microdonnées :** sous l’accordéon `Options d'exécution avancées` sur l’onglet `Exécutez le modèle`, une 
  bascule de sortie de microdonnées a été ajoutée pour que les utilisateurs du modèle puissent choisir d'exporter les microdonnées 
  dans la base de données ou dans un fichier texte (format .csv).
 - **Style :** la palette de couleurs a été mise à jour, de nouvelles icônes ont été ajoutées et plus d’espace
  a été placé entre les éléments sélectionnés.
  
### Modèles de jouets

Un nouveau modèle de jouet, `BMIGen`, est désormais livré avec `stcopenmpp`. Ce modèle est modèle de microsimulation portant sur 
l’indice de masse corporelle et la mortalité. `BMIGen` est une version réduite de la 
[Modèle de santé de la population (`POHEM`)](https://www.statcan.gc.ca/en/microsimulation/index#pohem), 
qui est un modèle de microsimulation portant sur certaines des maladies chroniques et des certains facteurs de risque dans lequel 
l’individu est l’unité d’analyse.

### Mises à jour des plugins groupés

- **auditr** : mis à niveau vers la v0.33.
- **openmpp :** mis à niveau vers la v0.0.2.9000.
- **inst_ubuntu :** version initiale ajoutée. Ce plugin fournit des scripts shell pour installer la distribution 
  OpenM++ de Statistique Canada (`stcopenmpp`) sur Ubuntu, ainsi que le compilateur   la chaîne d’outils et les outils de construction 
  dont il a besoin.
  
### Documentation
 
Un nouveau guide [Building from source](docs/building-from-source) a été ajouté au dossier `docs` avec 
des instructions comment compiler manuellement `stcopenmpp` à partir des sources sous Linux et sous Windows.

</details>

# `stcopenmpp` 0.2 Release Notes/Notes de Mise à Jour

<details close>

<summary>English</summary>

### New or updated features
* **Open M++:** synched to OpenM++ 1.17.13 (https://github.com/openmpp/main/releases/tag/v1.17.13), which is a recent security patch release. A special thank you to **Anatoly Cherkassky** for all his work on this release:
	* **C++ core:** security fix: remove polyfill.io from model documentation.
	* **C++ core:** improve error handling if number of sub-values exceeding number of random streams.
	* **C++ core:** fix an possible MPI model deadlock if number of CPU cores exceeds number of sub-values.
	* **C++ and Go:** enforce SQLite database referential integrity.
	* **Go:** fix sql error if microdata entities group deleted from database.
	* **Go:** support native Go database drivers for MySQL, PostgreSQL, MS SQL.
	* **Go and UI:** read-only model publishing, users can view models but not run or update.
	* **UI:** in table info dialog show expression dimension label only if notes are not empty.
* **`stcopenmpp` builds:** upgraded build dependencies (7-Zip, Go, Node.js).
* **`stcopenmpp` builds:** added an Ubuntu 22.04 build.
* **UI:** updated default path, url and router mode so the links are not broken when launching the UI in a Kubeflow environment.
* **UI:** changed header text and link to `stcopenmpp` branding.
* **UI:** changed primary color and favicon bundle to `stcopenmpp` branding.
* **UI:** fixed broken model documentation links for parameter notes.
* **Model run log:** changed a couple of labels in the model run log for clarity.
* **Plugins:** upgraded Matt Warkentin's **openmpp** R package to v0.0.2.

</details>

<details close>

<summary>Français</summary>

### Fonctionnalités nouvelles ou mises à jour

* **OpenM++ :** synchronisé avec OpenM++ 1.17.13 (https://github.com/openmpp/main/releases/tag/v1.17.13), qui est une version récente du correctif de sécurité.. Un merci spécial à **Anatoly Cherkassky** pour tout son travail sur cette sortie :
	* **C++ core :** correctif de sécurité : supprimer polyfill.io de la documentation du modèle.
	* **C++ core :** amélioration de la gestion des erreurs si le nombre de sous-valeurs dépasse le nombre de flux aléatoires.
	* **C++ core :** correction d’un blocage possible du modèle MPI si le nombre de CPU cores dépasse le nombre de sous-valeurs.
	* **C++ et Go :** renforcent l’intégrité référentielle de la base de données SQLite.
	* **Go :** corriger l’erreur sql si le groupe d’entités de microdonnées est supprimé de la base de données.
	* **Go :** prend en charge les pilotes natifs de base de données Go pour MySQL, PostgreSQL, MS SQL.
	* **Go et UI :** publication de modèles en lecture seule, les utilisateurs peuvent visualiser les modèles mais pas les exécuter ou les mettre à jour.
	* **UI :** dans la boîte de dialogue d’informations de la table, afficher l’étiquette de dimension d’expression uniquement si les notes ne sont pas vides.
* **Compilation d'`stcopenmpp` :** mise à jour des dépendances de build (7-Zip, Go, Node.js).
* **Compilation d'`stcopenmpp` :** ajout d’une version Ubuntu 22.04.
* **UI :** mise à jour du chemin par défaut, de l’URL et du mode routeur afin que les liens ne soient pas rompus lors du lancement de l’interface utilisateur dans un environnement Kubeflow.
* **UI :** changement du texte d’en-tête et du lien vers la marque `stcopenmpp`.
* **UI :** changement de la couleur primaire et du paquet favicon en marque `stcopenmpp`.
* **UI :** correction des liens de documentation du modèle cassés pour les notes de paramètres.
* **Journal d’exécution du modèle :** modification de quelques étiquettes dans le journal d’exécution du modèle pour plus de clarté.
* **Plugins :** mise à jour du paquet R de Matt Warkentin **openmpp** vers la v0.0.2.

</details>

# `stcopenmpp` 0.1 Release Notes/Notes de Mise à Jour

<details close>
<summary>English</summary>

### New or updated features
* **Synched to Open M++:** `stcopenmpp` is synched to the latest release of OpenM++ (https://github.com/openmpp/main/releases).
* **Repo structure:** OpenM++ source code and documentation are housed across 10 different repos. `stcopenmpp` condenses nine of these repos (excluding the “githubwikito-converter” repo) into subdirectories within a new “ompp” directory. These subdirectories are manually synced with any changes to the original repos via Git’s subtree feature. In addition to the “ompp” directory, there are also “plugins” and “docs” directories at the root level of the `stcopenmpp` repo.
* **ompp: ompp-docker** sub-folders were restructured and renamed. Dockerfiles in the build folders were updated to reflect the new repo structure. Dependencies were updated and Artifactory integration was added.
* **plugins: auditr** is an experimental cross-platform (Windows and Linux) R package that provides R6 methods for auditing two releases of OpenM++/`stcopenmpp`. The methods in this package allow users to download and extract two releases of `stcopenmpp`, clone and checkout a model from any point in its history, compile the model asynchronously in both releases, run the model asynchronously in both releases and generate an audit report in Quarto (https://github.com/statcan/stcopenmpp/tree/main/plugins/auditr).
* **plugins: compare-two-ompp-versions** is a Python tool for automating `stcopenmpp`/OpenM++ model testing and comparison. This makes it easy to test your models across different OpenM++ versions and see exactly what changed. Originally built by **Parsa Abadi** (first version available on [GitHub](https://github.com/parsaabadi/ompp-testing-automation/tree/main)), and maintained/developed further by Statistics Canada's Health Analysis and Modelling Division (https://github.com/statcan/stcopenmpp/tree/main/plugins/compare-two-omppversions).
* **plugins: openmpp** is an R package developed by **Matt Warkentin**. The goal of this package is to provide a programmatic interface to the OpenM++ API directly from R, to simplify creating scenarios, running models, and gathering results for further processing (https://github.com/statcan/stcopenmpp/tree/main/plugins/openmpp).
* **ompp/ompp-r: openMpp** is the original R package built for interacting with OpenM++. It allows users to read and update OpenM++ model databases on a local machine. This package, which is primarily used internally at Statistics Canada, has been updated to v0.8.8 to be used with R version 4. Some additional functions have been included, and function documentation has been modernized and updated so that it can be regenerated automatically by roxygen2 whenever the package is rebuilt. We recommend the use of the more updated openmpp package, described above.
### Bug fixes and minor improvements
* **Utilities:** fixed column misalignment and cascading data corruption in csvformatted model run lists when using **dbget.exe** and **dbcopy.exe**.
* **UI:** removed orphaned output table expression description header that appears in
the output table information modal.
### Download code and binaries for the following distributions
* Windows
* Windows MPI
* Linux Debian
* Linux Debian MPI
* Linux Ubuntu
* Linux Ubuntu MPI
</details>


<details close>
<summary>Français</summary>

### Fonctionnalités nouvelles ou mises à jour
* **Synched to Open M++:** `stcopenmpp` est synchronisé avec la dernière version d’OpenM++ (https://github.com/openmpp/main/releases).
* **Structure des dépôts:** le code source et la documentation OpenM++ sont hébergés dans 10 dépôts différents. `stcopenmpp` condense neuf de ces dépôts (à l’exception du dépôt « github-wikito-converter ») en sous-répertoires au sein d’un nouveau répertoire « ompp ». Ces sous-répertoires sont synchronisés manuellement avec tout changement apporté aux dépôts originaux à partir de la fonction sous-arbre de Git. En plus du répertoire « ompp », il existe aussi des répertoires « plugins » et « docs » au niveau racine du dépôt.
* Les sous-dossiers **ompp: ompp-docker** ont été restructurés et renommés. Les fichiers Dockerfiles figurant dans les dossiers de compilation ont été mis à jour pour représenter la nouvelle structure des dépôts. Les dépendances ont été mises à jour et l’intégration d’Artifactory a été ajoutée.
* **plugins: auditr** est un paquet R expérimental multiplateforme (Windows et Linux) qui fournit des méthodes R6 pour auditer deux versions d’OpenM++/`stcopenmpp`. Les méthodes de ce paquet permettent aux utilisateurs de télécharger et d’extraire deux versions de `stcopenmpp`, de cloner et de consulter un modèle à n’importe quel point de son historique, de compiler le modèle et de l’exécuter de manière asynchrone dans les deux versions, et de générer un rapport d’audit dans Quarto (https://github.com/statcan/stcopenmpp/tree/main/plugins/auditr).
* **plugins: compare-two-ompp-versions** est un outil Python permettant d’automatiser les tests et la comparaison de modèles `stcopenmpp`/OpenM++. Il vous permet de tester facilement vos modèles sur différentes versions d’OpenM++ et de voir exactement ce qui a changé. Il a été initialement développé par **Parsa Abadi** (première version accessible sur [GitHub](https://github.com/parsaabadi/ompp-testing-automation/tree/main)), puis maintenu et perfectionné par la Division de l’analyse et de la modélisation de la santé (https://github.com/statcan/stcopenmpp/tree/main/plugins/compare-two-ompp-versions).
* **plugins: openmpp** est un paquet R développé par **Matt Warkentin**. L’objectif de ce paquet est de fournir une interface programmatique à l’API OpenM++ directement depuis R, afin de simplifier la création de scénarios, l’exécution de modèles et la collecte de résultats pour un traitement ultérieur (https://github.com/statcan/stcopenmpp/tree/main/plugins/openmpp).
* **ompp/ompp-r: openMpp** est le paquet R original conçu pour interagir avec OpenM++. Il permet aux utilisateurs de lire et de mettre à jour des bases de données modèles OpenM++ sur un poste de l’utilisateur. Ce paquet, principalement utilisé à l’interne à Statistique Canada, a été mis à jour pour passer à la version 0.8.8 et être utilisé avec la version 4 de R. Certaines fonctions ont été ajoutées et la documentation des fonctions a été modernisée et mise à jour afin qu’elle puisse être régénérée automatiquement par roxygen2 chaque fois que le paquet est reconfiguré. Nous recommandons l’utilisation du paquet openmpp plus récent, décrit ci-dessus.

### Corrections de bogues et améliorations mineures
* **Utilitaires:** correction du désalignement des colonnes et de la corruption des données en cascade dans les listes d’exécution de modèles en format csv lors de l’utilisation de **dbget.exe** et **dbcopy.exe**.
* **Interface utilisateur:** suppression de l’en-tête de description de l’expression de la table de sortie orpheline qui s’affiche dans le modal d’information de la table de sortie.

### Code de téléchargement et binaires pour les répartitions suivantes
* Windows
* Windows MPI
* Linux Debian
* Linux Debian MPI
* Linux Ubuntu
* Linux Ubuntu MPI
</details>