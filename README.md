# stcopenmpp

[![Lifecycle:
stable](https://img.shields.io/badge/lifecycle-stable-brightgreen.svg)](https://lifecycle.r-lib.org/articles/stages.html#stable)

<details open>

<summary>English</summary>

`stcopenmpp` is a **fork** of <a href="https://github.com/openmpp/main" target="_blank">OpenM++</a>, released for Statistics Canada’s internal microsimulation modellers as well as external government and academic partners.

**OpenM++** is an open-source framework designed for building and running complex microsimulation models, developed and maintained by **Steve Gribble** and **Anatoly Cherkassky**. 
OpenM++, inspired by its predecessor Modgen, offers a modern, portable, scalable, and cross-platform product that supports dynamic microsimulation, allowing users to model life events such as aging, employment, or health transitions. 
With the core functionality written in C++, OpenM++ includes a compiler, runtime engine, and user interface, and integrates with languages like R and Python for data analysis and visualization through OpenM++ API endpoints. 
The API is part of the OpenM++ Web Service (OMS) and is designed to support tasks such as configuring models, running simulations, and retrieving results. 
The modular design of OpenM++ enables flexible model development, and it can be deployed on desktops or in cloud environments. 

## Orientation to this repo

The `stcopenmpp` repo contains three folders: **ompp**, **plugins** and **docs**. The **ompp** folder contains the main body of the code. The **plugins** folder contains open-source tools developed to make testing and interacting with `stcopenmpp` easier. 
The **docs** folder brings forward and curates useful documentation for model developers and other interested readers.

## What is microsimulation?

**Microsimulation** is a technique to model how individual people or households might be affected by changes in factors like taxes, health policies, or social policies and programs. 
Instead of looking at averages or population groups, it uses detailed data about individuals—like age, income, or health—to simulate what could happen to each person under different scenarios. 
For example, if the government changes a health policy, microsimulation can estimate how that change would affect individuals across the country. It can be used to look at one moment in time or to project changes over many years. 
This technique can help policymakers understand the real-life impact of decisions and design fairer, more effective policies. 
Please see the following links for a more comprehensive overview of the technique prepared by **Martin Spielauer** and [linked here](https://github.com/statcan/stcopenmpp/tree/main/docs/a-discussion-of-microsimulation-by-the-author-of-riskpaths). Interested readers should also explore the International Microsimulation Association website and journal <a href="https://www.microsimulation.org" target="_blank">International Microsimulation Association</a>.

## Applications of microsimulation modelling at Statistics Canada

Starting in the 1990s, Statistics Canada has developed microsimulation models of health, demography and retirement income. 
These models have been used by decision makes in federal and provincial governments, as well as academics, to investigate the future characteristics of the Canadian population and explore the impacts of policy or program directions. 
If you would like to know more about microsimulation applications developed at Statistics Canada, please check out the following <a href="https://www.statcan.gc.ca/en/microsimulation/index" target="_blank">website</a> or send us an email at: **[microsimulation@statcan.gc.ca](mailto:microsimulation@statcan.gc.ca)**

## Download and install

To install `stcopenmpp`, download the zip file and extract the source code and binaries, appropriate for your operating system, to the location of your choice. The latest release of `stcopenmpp` can be found [here](https://github.com/statcan/stcopenmpp/releases). **Pre-compiled binaries** are available for Windows and Linux (Debian and Ubuntu).

**Important note**: Pre-compiled binaries for MacOS will no longer be released as part of `stcopenmpp`. <a href="https://github.com/openmpp/main" target="_blank">OpenM++</a> provides pre-compiled binaries for MacOS.

## Steps for running `stcopenmpp` in Windows

1. Using Windows File Explorer, enter or browse to the `stcopenmpp` directory location and navigate to the bin folder. 

2. Double-click the launch-ui-windows.bat file (ompp_ui.bat also works). This will start the process responsible for running the OMS. Note that the local host address — including the port number— will be printed in the console. 

3. Click on the address to open the web-based user interface (UI). Any models located in models/bin will be listed in the UI.

4. Select your model and set up a run. 

**Important note:** If OM_ROOT was previously set as a permanent environment variable and if it differs from the current `stcopenmpp` location, it will need to be changed prior to launching the OMS. Use the following command in the .cmd window:

`**setx OM_ROOT 'C:/path/to/your/current/stcopenmpp/instance'**`


## Steps for running `stcopenmpp` in Linux

1. If using Linux in a desktop environment (e.g., Debian GNU/Linux 13 “Trixie” with GNOME), enter the `stcopenmpp` directory location using Files and navigate to the bin folder. 

2. Right click start_oms.sh and select the Run as a Program option. This will start the process responsible for running the OMS. Note that the local host address — including the port number— will be printed in the console. 

3. Click on the address to open the web-based user interface (UI). Any models located in models/bin will be listed in the UI.

4. Select your model and set up a run. 

**Important note:** If OM_ROOT was previously set as a permanent environment variable and if it differs from the current `stcopenmpp` location, it will need to be changed prior to launching the OMS. Use the following command in the Linux console: 

`**export OM_ROOT=/home/path/to/your/current/stcopenmpp/instance**`

## Using `stcopenmpp` with built-in models

`stcopenmpp` is bundled with several test/toy models, contained in the ‘models’ folder of the downloaded file. One such model is [RiskPaths](https://github.com/statcan/stcopenmpp/tree/main/ompp/models/RiskPaths). RiskPaths is a simple, competing risk, case-based continuous time microsimulation model. Its main use is as a teaching tool, introducing microsimulation to practitioners and demonstrating how dynamic microsimulation models can be efficiently programmed using OpenM++. 
RiskPaths has been extensively documented and using this model is a great way to get started with microsimulation modelling. 
We recommend you check out the following links within the repo and the wiki for more information on getting started with RiskPaths.

- **Repo:** <a href="https://github.com/statcan/stcopenmpp/tree/main/ompp/models/RiskPaths/doc" target="_blank">github.com/statcan/stcopenmpp/tree/main/ompp/models/RiskPaths/doc</a>
- **Wiki:** <a href="https://github.com/openmpp/openmpp.github.io/wiki" target="_blank">github.com/openmpp/openmpp.github.io/wiki</a>

## Using `stcopenmpp` with your existing models  

If you are a model developer at Statistics Canada or in one of our partner organizations, you likely already have a model you are working on. To get started with `stcopenmpp`:  

1. Download the software to the location of your choice, as described [above](#download-and-install). 

2. Then open the existing model solution file (e.g., RiskPaths-ompp.sln) in Visual Studio Community or Professional.

3. “Rebuild Solution” from the “Build” menu option. 

4. After the model has been successfully built, if the OMS does not start automatically, double-click the “start-ompp-ui.bat” file to launch the service and the UI. 

**Important notes:** If you use an IDE to compile, we recommend using Visual Studio Community or Professional 2022 or 2026. Additionally, it is important to note that the model should be compiled in the same version of Visual Studio in which it is built, otherwise additional set-up steps will be required: instructions are <a href="https://github.com/openmpp/openmpp.github.io/wiki/Windows-Quick-Start-Developer#using-older-versions-of-visual-studio" target="_blank">linked here</a>.
 
## Running your existing models with `stcopenmpp`

There are a couple of ways to run existing microsimulation models developed with `stcopenmpp`. 

**If you like using a UI-**
- Use the built-in user interface (UI) that comes with `stcopenmpp`. See the [section above](#steps-for-running-stcopenmpp-in-windows) to learn how to get the webservice started and open the UI. The UI is documented extensively <a href="https://github.com/openmpp/openmpp.github.io/wiki" target="_blank">here</a>.

**If you like running your models programmatically-**
- We recommend using the `openmpp` R package developed by **Matt Warkentin** that comes bundled with `stcopenmpp`, see the [plugins](https://github.com/statcan/stcopenmpp/tree/main/plugins/openmpp) folder. 

## Developing new models with `stcopenmpp`

Developing new microsimulation or agent-based models with `stcopenmpp` is beyond the scope of this readme. However, the RiskPaths documentation, [linked above](#using-stcopenmpp-with-built-in-models), provides an excellent introduction to model development. 
More detailed information on model development topics can be found on the <a href="https://github.com/openmpp/openmpp.github.io/wiki/Model-Development-Topics" target="_blank">OpenM++ wiki</a>. In addition, the Modgen (the predecessor of OpenM++) developers guide remains an important resource for developing OpenM++- based models. We have [linked](https://github.com/statcan/stcopenmpp/tree/main/docs/modgen-developers-guide) it in the repo for your convenience.

## Documentation

**Steve Gribble** and **Anatoly Cherkassky** have developed comprehensive documentation for OpenM++ that is available <a href="https://github.com/openmpp/openmpp.github.io/wiki" target="_blank">here</a>.

`stcopenmpp` will be documented in this readme and through release notes on this repo.  

## Contributor Guidelines

Contributions to this package are welcome, however contributions and issues from Statistics Canada modellers and collaborators will be given priority. The preferred method of contribution is through a pull request. 

Before contributing, please create an issue to alert the project team. More details on contributing can be found in the CONTRIBUTING document.


</details>


<hr style="border: 3px solid grey; height: 4px">


<details open>

<summary>Français</summary>

Le logiciel `stcopenmpp` est un **embranchement** de <a href="https://github.com/openmpp/main" target="_blank">OpenM++</a>; il a été conçu pour les concepteurs de modèles de microsimulation de Statistique Canada et les partenaires gouvernementaux et universitaires externes. 

**OpenM++** est un cadre à code source ouvert conçu pour l’élaboration et l’exécution de modèles de microsimulation complexes; son développement et sa maintenance sont assurés par **Steve Gribble** et **Anatoly Cherkassky**. 
OpenM++, qui s’inspire de son prédécesseur Modgen, est un produit moderne, portable, évolutif et multiplateforme qui supporte la microsimulation dynamique et permet aux utilisateurs de modéliser des événements de vie tels que le vieillissement, l’emploi ou les changements à l’échelle de la santé. Doté d’une fonctionnalité de base écrite en C++, OpenM++ comprend un compilateur, un moteur d’exécution et une interface utilisateur, le tout s’intégrant avec des langages comme R et Python pour faciliter l’analyse et la visualisation de données grâce à ses points de terminaison API. L’interface API fait partie du service Web OpenM++ (OMS) et est conçue pour réaliser des tâches telles que la configuration de modèles, l’exécution de simulations et la récupération de résultats. La conception modulaire d’OpenM++ permet un développement de modèles flexible et la plateforme peut être déployée dans les ordinateurs de bureau ou dans des environnements infonuagiques. 

## Orientation du présent dépôt 

Ce dépôt contient trois dossiers : **ompp**, **plugins** et **docs**. Le dossier ompp présente les éléments essentiels du code. Le dossier **plugins** contient des logiciels ouverts développés pour faciliter les tests et l’interaction avec `stcopenmpp`. Le dossier **docs** présente et maintient la documentation utile pour les développeurs de modèles et les autres lecteurs intéressés. 

## Qu’est-ce que la microsimulation? 

La **microsimulation** est une technique permettant de modéliser la manière dont les personnes ou les ménages pourraient être touchés par des changements dans des facteurs tels que les impôts, les politiques de santé ou les programmes sociaux. Au lieu d’examiner les moyennes ou les groupes de population, elle utilise des données détaillées sur les personnes — comme l’âge, le revenu ou l’état de santé — pour simuler ce qui pourrait arriver à chaque cas dans différents scénarios. Si, par exemple, le gouvernement modifie une politique de santé, la microsimulation peut estimer comment ce changement toucherait les citoyens du pays tout entier. On peut aussi l’utiliser pour visualiser la situation à un moment donné ou suivre des changements sur plusieurs années. Cette technique peut aider les décisionnaires à comprendre l’impact réel des décisions et à concevoir des politiques plus justes et efficaces. Pour obtenir un aperçu plus complet préparé par **Martin Spielauer**, veuillez cliquer sur [les liens suivants](https://github.com/statcan/stcopenmpp/tree/main/docs/a-discussion-of-microsimulation-by-the-author-of-riskpaths). Les lecteurs intéressés devraient également consulter le site Web et la revue de <a href="https://www.microsimulation.org" target="_blank">l’International Microsimulation Association</a>. 

## Applications de la modélisation par microsimulation à Statistique Canada

Depuis les années 1990, Statistique Canada a développé des modèles de microsimulation en matière de santé, de démographie et de revenus de retraite (lien vers le nouveau site Web). Ils ont été utilisés par les décisionnaires des gouvernements fédéral et provinciaux, ainsi que par le milieu universitaire, pour étudier les caractéristiques futures de la population canadienne et mieux connaître les répercussions des politiques ou des programmes. Si vous souhaitez en savoir plus sur les applications de microsimulation développées à Statistique Canada, veuillez consulter le <a href="https://www.statcan.gc.ca/fr/microsimulation/index" target="_blank">site Web</a> suivant ou nous envoyer un courriel à l’adresse : **[microsimulation@statcan.gc.ca](mailto:microsimulation@statcan.gc.ca)**. 

## Téléchargement et installation

Pour installer `stcopenmpp`, téléchargez le fichier compressé et extrayez le code source et les jeux de binaires appropriés selon votre système d’exploitation, en vue de les enregistrer à l’emplacement de votre choix. La dernière version de `stcopenmpp` est disponible [ici](https://github.com/statcan/stcopenmpp/releases). Des **jeux de binaires précompilés** sont disponibles pour Windows et Linux (Debian et Ubuntu). **Note importante:** Veuillez noter que les jeux de binaires précompilés pour MacOS ne seront pas disponibles dans `stcopenmpp`. <a href="https://github.com/openmpp/main" target="_blank">OpenM++</a> fournit des jeux de binaires précompilés pour MacOS.

## Étapes pour exécuter `stcopenmpp` dans Windows

1. Entrez l’emplacement du répertoire `stcopenmpp` dans l’Explorateur de fichiers de Windows et naviguez jusqu’au dossier « bin ».

2. Double-cliquez sur launch-ui-windows.bat (ompp_ui.bat fonctionne aussi), ce qui lancera le processus d’exécution du service OMS. Notez que l’adresse locale de l’hôte — y compris le numéro de port — sera imprimée dans la console. 

3. Cliquez sur l’adresse pour ouvrir l’interface utilisateur Web. Tous les modèles figurant dans les dossiers modèles ou « bin » seront énumérés dans l’interface utilisateur. 

4. Sélectionnez votre modèle et lancez une simulation. 

**Note importante:** Si OM_ROOT était auparavant défini comme variable d’environnement permanente et qu’il diffère de l’emplacement actuel de `stcopenmpp`, il devra être modifié avant le lancement du service OMS. Utilisez la commande suivante dans la console Linux : 

`**setx OM_ROOT 'C:/path/to/your/current/stcopenmpp/instance'**`

## Étapes pour exécuter `stcopenmpp` dans Linux

1. Si vous utilisez Linux dans un environnement de bureau (par exemple Debian GNU/Linux 13 « Trixie » avec GNOME), entrez l’emplacement du répertoire `stcopenmpp` dans Fichiers et naviguez jusqu’au dossier « bin ». 

2. Faites un clic droit sur start_oms.sh et sélectionnez l’option Exécuter en tant que programme, ce qui lancera le processus d’exécution du service OMS. Notez que l’adresse locale de l’hôte — y compris le numéro de port — sera imprimée dans la console. 

3. Cliquez sur l’adresse pour ouvrir l’interface utilisateur Web. Tous les modèles figurant dans les dossiers modèles ou « bin » seront énumérés dans l’interface utilisateur. 

4. Sélectionnez votre modèle et lancez une simulation. 

**Note importante:** Si OM_ROOT était auparavant défini comme variable d’environnement permanente et qu’il diffère de l’emplacement actuel de `stcopenmpp`, il devra être modifié avant le lancement du service OMS. Utilisez la commande suivante dans la console Linux : 

`**export OM_ROOT=/home/path/to/your/current/stcopenmpp/instance**`

## Utilisation de `stcopenmpp` avec des modèles intégrés

Le logiciel `stcopenmpp` comprend plusieurs modèles d’essai et modèles-jouets, que l’on retrouve dans le dossier « models » du fichier téléchargé. [RiskPaths](https://github.com/statcan/stcopenmpp/tree/main/ompp/models/RiskPaths) figure parmi ces modèles. 
RiskPaths est un modèle de microsimulation simple en temps continu, basé sur les cas et sur le risque concurrentiel. Il constitue principalement un outil pédagogique, présentant la microsimulation aux praticiens et démontrant comment les modèles dynamiques de microsimulation peuvent être programmés efficacement à l’aide d’OpenM++. RiskPaths représente une excellente façon de se lancer dans la modélisation par microsimulation. Nous vous recommandons de consulter les liens suivants, figurant dans le dépôt et dans le wiki, pour en savoir plus sur le démarrage dans RiskPaths. 

- **Dépôt:** <a href="https://github.com/statcan/stcopenmpp/tree/main/ompp/models/RiskPaths/doc" target="_blank">github.com/statcan/stcopenmpp/tree/main/ompp/models/RiskPaths/doc</a>
- **Wiki:** <a href="https://github.com/openmpp/openmpp.github.io/wiki" target="_blank">github.com/openmpp/openmpp.github.io/wiki</a> 

## Utilisation de `stcopenmpp` dans vos modèles existants

En votre qualité de développeur de modèles à Statistique Canada ou dans l’une de nos organisations partenaires, il est probable que vous travailliez déjà sur un modèle. Pour commencer à utiliser `stcopenmpp` :   

1. Téléchargez le logiciel à l'emplacement de votre choix, comme décrit [ci-dessus](#téléchargement-et-installation). 

2. Il faut ensuite ouvrir le fichier de la solution modèle (par exemple, RiskPaths-ompp.sln) dans Visual Studio Community ou Professional. 

3. Dans l’option de menu « Build », sélectionner « Rebuild Solution ». 

4. Une fois le modèle construit avec succès, si le service OMS ne démarre pas automatiquement, double-cliquez sur le fichier « start-ompp-ui.bat » pour lancer le service et l’interface utilisateur. 

**Notes importantes :** Si vous utilisez un environnement de développement intégré pour la compilation, nous recommandons d’utiliser Visual Studio Community ou Professional 2022 ou 2026. Il est important de noter que le modèle doit être compilé dans la même version de Visual Studio qui a été utilisée pour le construire, sinon des étapes de configuration supplémentaires seront nécessaires; les instructions sont accessibles <a href="https://github.com/openmpp/openmpp.github.io/wiki/Windows-Quick-Start-Developer#using-older-versions-of-visual-studio" target="_blank">ici</a>. 

## Traitement de modèles existants dans `stcopenmpp`

Les modèles de microsimulation existants développés avec `stcopenmpp` peuvent être exécutés de différentes manières. 

**Si vous aimez travailler avec une interface utilisateur-**
- utilisez l’interface utilisateur intégrée fournie avec `stcopenmpp`. Consultez la [section ci-dessus](#exécution-de-stcopenmpp-dans-windows) pour savoir comment lancer le service Web et ouvrir l’interface utilisateur. L’interface utilisateur est présentée <a href="https://github.com/openmpp/openmpp.github.io/wiki" target="_blank">en détail ici</a>. 

**Si vous aimez exécuter vos modèles comme dans un programme-**
- nous vous recommandons d’utiliser l’ensemble openmpp pour R développé par **Matt Warkentin** qui est fourni avec `stcopenmpp`; voir le dossier [plugins](https://github.com/statcan/stcopenmpp/tree/main/plugins/openmpp). 

## Élaboration de nouveaux modèles avec `stcopenmpp`

L’élaboration de nouveaux modèles de microsimulation ou de modèles orientés agent avec `stcopenmpp` dépasse le cadre de notre présentation. La documentation RiskPaths, ([lien ci-dessus](#si-vous-aimez-exécuter-vos-modèles-comme-dans-un-programme)) contient une excellente introduction au développement de modèles. Pour en savoir plus sur le développement de modèles, veuillez consulter la page <a href="https://github.com/openmpp/openmpp.github.io/wiki/Model-Development-Topics" target="_blank">wiki d’OpenM++</a>. De plus, le guide pour développeurs Modgen (prédécesseur d’OpenM++) demeure une ressource importante pour l’élaboration de modèles basés sur OpenM++. Nous avons prévu des [liens](https://github.com/statcan/stcopenmpp/tree/main/docs/modgen-developers-guide) dans ce document, pour faciliter la recherche. 

## Documentation

**Steve Gribble** et **Anatoly Cherkassky** ont rassemblé une documentation complète sur OpenM++ (cliquez <a href="https://github.com/openmpp/openmpp.github.io/wiki" target="_blank">ici</a>). Le logiciel `stcopenmpp` fera l’objet d’une documentation dans le présent document et dans les notes de version du dépôt. 

## Lignes directrices pour les contributeurs

Vous pouvez en tout temps ajouter une contribution à l’ensemble ou poser des questions à ce sujet, mais celles qui proviennent de modélisateurs et de collaborateurs de Statistique Canada sont prioritaires. La méthode préférée, pour les contributions, consiste en une demande de fusion. Au préalable, veuillez créer un problème pour alerter l’équipe du projet. On trouvera plus de précisions à ce sujet dans le document intitulé « CONTRIBUTIONS ». 

</details>