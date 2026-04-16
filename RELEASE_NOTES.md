# `stcopenmpp` 0.1 Release Notes/Notes de Mise à Jour

<details open>
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


<details open>
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