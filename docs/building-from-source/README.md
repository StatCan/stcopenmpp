# Building from source


# 1. Upstream sync with OpenM++

OpenM++ uses
<a href="https://github.com/orgs/openmpp/repositories" target="_blank">several
repos</a> during the build process whereas stcopenmpp uses a single repo
containing seven of OpenM++’s repos. The **main** repo in OpenM++ is in
stcopenmpp’s **ompp** folder, the **go** repo is in **ompp/ompp-go**,
the **UI** repo is in **ompp/ompp-ui**, etc. The
**github-wikito-converter**, **mac** and **openmpp.github.io** repos are
not included in stcopenmpp.

If there are no OpenM++ updates, this step can be ignored. Otherwise,
the approach to syncing stcopenmpp with OpenM++ involves setting up a
Git subtree on an empty branch for each updated OpenM++ repo and merging
into the stcopenmpp development branch (**v0.x** in the examples below).
The subtree setup is a one-time action.

> **Note:** there will often be conflicts. In such cases, Git Gui and VS
> Code are useful for quickly visualizing and accepting/rejecting
> changes.

## Main repo

``` bash
# Set variables
SUBTREE_BRANCH=ompp-main
DEV_BRANCH=v0.x
REPO_URL=https://github.com/openmpp/main.git
REMOTE_BRANCH=master
SUBTREE_PREFIX=ompp

# Fetch upstream and record its latest commit hash
git fetch "$REPO_URL" "$REMOTE_BRANCH"
UPSTREAM_HASH=$(git rev-parse FETCH_HEAD)
UPSTREAM_SHORT=$(git rev-parse --short FETCH_HEAD)

# Try to add empty branch
if git checkout --orphan "$SUBTREE_BRANCH" 2>/dev/null; then
  git rm -rf .
  git commit --allow-empty -m "Make initial commit"
  git subtree add --prefix="$SUBTREE_PREFIX" FETCH_HEAD
else
  git checkout "$SUBTREE_BRANCH"
  # Merge the exact commit we fetched
  git subtree merge --prefix="$SUBTREE_PREFIX" FETCH_HEAD
fi

# Push any changes
git push

# Move to dev branch
git checkout "$DEV_BRANCH"

# Merge branch
git merge --no-ff --allow-unrelated-histories "$SUBTREE_BRANCH"

# Resolve any conflicts
# ...

# Commit changes
git commit -m "Sync OpenM++ main repo up to $UPSTREAM_HASH"

# Push changes
git push
```

## Go repo

``` bash
# Set variables
SUBTREE_BRANCH=ompp-go
DEV_BRANCH=v0.x
REPO_URL=https://github.com/openmpp/go.git
REMOTE_BRANCH=master
SUBTREE_PREFIX=ompp-go

# Fetch upstream and record its latest commit hash
git fetch "$REPO_URL" "$REMOTE_BRANCH"
UPSTREAM_HASH=$(git rev-parse FETCH_HEAD)
UPSTREAM_SHORT=$(git rev-parse --short FETCH_HEAD)

# Try to add empty branch
if git checkout --orphan "$SUBTREE_BRANCH" 2>/dev/null; then
  git rm -rf .
  git commit --allow-empty -m "Make initial commit"
  git subtree add --prefix="$SUBTREE_PREFIX" FETCH_HEAD
else
  git checkout "$SUBTREE_BRANCH"
  # Merge the exact commit we fetched
  git subtree merge --prefix="$SUBTREE_PREFIX" FETCH_HEAD
fi

# Push any changes
git push

# Move to dev branch
git checkout "$DEV_BRANCH"

# Merge branch
git merge --no-ff --allow-unrelated-histories "$SUBTREE_BRANCH"

# Resolve any conflicts
# ...

# Commit changes
git commit -m "Sync OpenM++ go repo up to $UPSTREAM_HASH"

# Push changes
git push
```

## UI repo

``` bash
# Set variables
SUBTREE_BRANCH=ompp-ui
DEV_BRANCH=v0.x
REPO_URL=https://github.com/openmpp/UI.git
REMOTE_BRANCH=master
SUBTREE_PREFIX=ompp-ui

# Fetch upstream and record its latest commit hash
git fetch "$REPO_URL" "$REMOTE_BRANCH"
UPSTREAM_HASH=$(git rev-parse FETCH_HEAD)
UPSTREAM_SHORT=$(git rev-parse --short FETCH_HEAD)

# Try to add empty branch
if git checkout --orphan "$SUBTREE_BRANCH" 2>/dev/null; then
  git rm -rf .
  git commit --allow-empty -m "Make initial commit"
  git subtree add --prefix="$SUBTREE_PREFIX" FETCH_HEAD
else
  git checkout "$SUBTREE_BRANCH"
  # Merge the exact commit we fetched
  git subtree merge --prefix="$SUBTREE_PREFIX" FETCH_HEAD
fi

# Push any changes
git push

# Move to dev branch
git checkout "$DEV_BRANCH"

# Merge branch
git merge --no-ff --allow-unrelated-histories "$SUBTREE_BRANCH"

# Resolve any conflicts
# ...

# Commit changes
git commit -m "Sync OpenM++ UI repo up to $UPSTREAM_HASH"

# Push changes
git push
```

# 2. Build Docker images

These Docker build images — based on Dockerfiles in **ompp/ompp-docker**
(e.g., **debian-build**, **ubuntu-build**, **ubuntu22-build** and
**windows-build**) — are built and pushed to Docker Hub
(<a href="https://hub.docker.com/u/stcopenmpp" target="_blank">hub.docker.com/u/stcopenmpp</a>),
and used in the next step to build stcopenmpp from source.

## Debian image

``` bash
# Login to Docker Hub account
docker login

# Move into folder containing Dockerfile
cd /path/to/stcopnempp/ompp/ompp-docker/debian-build

# Build image
docker build -t stcopempp/stcopenmpp-build:debian .

# Push image to Docker Hub
docker push stcopenmpp/stcopenmpp-build:debian
```

## Ubuntu image

``` bash
# Login to Docker Hub account
docker login

# Move into folder containing Dockerfile
cd /path/to/stcopnempp/ompp/ompp-docker/ubuntu-build

# Build image
docker build -t stcopempp/stcopenmpp-build:ubuntu .

# Push image to Docker Hub
docker push stcopenmpp/stcopenmpp-build:ubuntu
```

## Ubuntu 22 image

> **Note:** this image is built to accomodate Linux servers at
> Statistics Canada running on Ubuntu 22.04 distributions. These servers
> can run pre-built models with the latest stcopenmpp Ubuntu build, but
> they cannot compile models with this build due to **glibc** (C runtime
> library) incompatibilities. Ubuntu 22 and Ubuntu 24 ship with
> different versions of **glibc**.

``` bash
# Login to Docker Hub account
docker login

# Move into folder containing Dockerfile
cd /path/to/stcopnempp/ompp/ompp-docker/ubuntu22-build

# Build image
docker build -t stcopempp/stcopenmpp-build:ubuntu22 .

# Push image to Docker Hub
docker push stcopenmpp/stcopenmpp-build:ubuntu22
```

## Windows image

``` bash
# Login to Docker Hub account
docker login

# Move into folder containing Dockerfile
cd /path/to/stcopnempp/ompp/ompp-docker/windows-build

# Build image
docker build -t stcopempp/stcopenmpp-build:windows .

# Push image to Docker Hub
docker push stcopenmpp/stcopenmpp-build:windows
```

# 3. Build stcopenmpp binaries

In this final step, the stcopenmpp binaries are built. These
pre-compiled files include the OpenM++ compiler, model run-time
libaries, the OpenM++ web service, compiled toy models, the UI and
utility tools. The binaries allow the user to use stcopenmpp without
having to compile anything.

## Debian

``` bash
# Set variables
$IMAGE_URL = "<url>/docker-remote/stcopenmpp/stcopenmpp-build:debian"
$GIT_URL = "https://github.com/statcan/stcopenmpp"
$BUILD_TAG = "v0.x"
$BUILD_CONFIGS = "RELEASE,DEBUG"
$MPI = ""
$FILE_NAME = "stcopenmpp-debian-v0.x"
$MOUNT_PATH = "<basepath>/ompp-builds/linux/debian"

# Pull image
docker pull ${IMAGE_URL}

# Build container
docker run `
  --rm `
  -it `
  -e OMPP_GIT_URL=${GIT_URL} `
  -e OMPP_BUILD_TAG=${BUILD_TAG} `
  -e OM_BUILD_CONFIGS=${BUILD_CONFIGS} `
  -e OM_MSG_USE=${MPI} `
  -e OMPP_DEPLOY_NAME=${FILE_NAME} `
  -v ${MOUNT_PATH}:/mnt/ompp-build `
  ${IMAGE_URL} `
  bash
```

## Ubuntu

``` bash
# Set variables
$IMAGE_URL = "<url>/docker-remote/stcopenmpp/stcopenmpp-build:ubuntu"
$GIT_URL = "https://github.com/statcan/stcopenmpp"
$BUILD_TAG = "v0.x"
$BUILD_CONFIGS = "RELEASE,DEBUG"
$MPI = ""
$FILE_NAME = "stcopenmpp-ubuntu-v0.x"
$MOUNT_PATH = "<basepath>/ompp-builds/linux/ubuntu"

# Pull image
docker pull ${IMAGE_URL}

# Build container
docker run `
  --rm `
  -it `
  -e OMPP_GIT_URL=${GIT_URL} `
  -e OMPP_BUILD_TAG=${BUILD_TAG} `
  -e OM_BUILD_CONFIGS=${BUILD_CONFIGS} `
  -e OM_MSG_USE=${MPI} `
  -e OMPP_DEPLOY_NAME=${FILE_NAME} `
  -v ${MOUNT_PATH}:/mnt/ompp-build `
  ${IMAGE_URL} `
  bash
```

## Ubuntu 22

``` bash
# Set variables
$IMAGE_URL = "<url>/docker-remote/stcopenmpp/stcopenmpp-build:ubuntu22"
$GIT_URL = "https://github.com/statcan/stcopenmpp"
$BUILD_TAG = "v0.x"
$BUILD_CONFIGS = "RELEASE,DEBUG"
$MPI = ""
$FILE_NAME = "stcopenmpp-ubuntu22-v0.x"
$MOUNT_PATH = "<basepath>/ompp-builds/linux/ubuntu22"

# Pull image
docker pull ${IMAGE_URL}

# Build container
docker run `
  --rm `
  -it `
  -e OMPP_GIT_URL=${GIT_URL} `
  -e OMPP_BUILD_TAG=${BUILD_TAG} `
  -e OM_BUILD_CONFIGS=${BUILD_CONFIGS} `
  -e OM_MSG_USE=${MPI} `
  -e OMPP_DEPLOY_NAME=${FILE_NAME} `
  -v ${MOUNT_PATH}:/mnt/ompp-build `
  ${IMAGE_URL} `
  bash
```

## Windows

``` bash
# Set variables
$IMAGE_URL = "<url>/docker-remote/stcopenmpp/stcopenmpp-build:windows"
$GIT_URL = "https://github.com/statcan/stcopenmpp"
$BUILD_TAG = "v0.x"
$BUILD_CONFIGS = "Release,Debug"
$BUILD_PLATFORMS = "Win32,x64"
$MPI = ""
$FILE_NAME = "stcopenmpp-windows-v0.x"
$MOUNT_PATH = "<basepath>\ompp-builds\windows"

# Pull image
docker pull ${IMAGE_URL}

# Build container
docker run `
  --rm `
  -it `
  -e OMPP_GIT_URL=${GIT_URL} `
  -e OMPP_BUILD_TAG=${BUILD_TAG} `
  -e OM_BUILD_CONFIGS=${BUILD_CONFIGS} `
  -e OM_BUILD_PLATFORMS=${BUILD_PLATFORMS} `
  -e OM_MSG_USE=${MPI} `
  -e OMPP_DEPLOY_NAME=${FILE_NAME} `
  -v ${MOUNT_PATH}:c:\build `
  ${IMAGE_URL}
```
