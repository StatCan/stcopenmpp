#!/bin/bash

# Pull/Clone repo
echo "GETTING ALL AVAILABLE VERSIONS OF OpenM++ ..."
cd ${HOME}
if [[ ! -d "${HOME}/stcopenmpp" ]];then
    git clone "https://github.com/StatCan/stcopenmpp"
    cd "${HOME}/stcopenmpp"
else
    cd "${HOME}/stcopenmpp"
    git pull
fi

# Set STC openmpp version
readarray -t myv < <(git tag)
echo "AVAILABLE VERSIONS:"
for ii in ${!myv[@]};do
    tmp=$(echo ${myv[${ii}]})
    echo "${ii} --> ${tmp}"
done;
echo ""
echo ""
itistheone=n
while [ ${itistheone} = n ];do
    read -p "Which version do you want to install (type in the number): " version_index < /dev/tty
    tmp=$(echo ${myv[${version_index}]})
    echo "The version to be installed is \"${tmp}\""
    read -p "Is this correct (y/n)?" itistheone < /dev/tty
    if [ ${itistheone} = y ];then
        break
    fi
done
myv=${myv[${version_index}]}
cd ${HOME}
rm -fr "${HOME}/stcopenmpp"
echo "Installing STC OpenM++ for Ubuntu ${myv}"
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# get Ubuntu version
uv=$(lsb_release -r)
uv=$(echo ${uv} | sed -e 's/^Release:[[:space:]]*\([0-9][0-9]\.[0-9][0-9].*\)$/\1/g')

echo "Installing STC OpenM++ for Ubuntu ${uv} ..."
if [[ ${uv} == "22.04" ]]; then
    myom=stcopenmpp-ubuntu22-${myv}
else
    myom=stcopenmpp-ubuntu-${myv}
fi
myurl=https://github.com/StatCan/stcopenmpp/releases/download/${myv}/${myom}-gh-release.tar.xz

echo "URL of version to be installed: ${myurl}"
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

cat "Downloading ${myom} ..."
cd ${HOME}
wget ${myurl}

cat "Extracting ${myom} ..."
tar -xvf ${myom}-gh-release.tar.xz
rm -fr ${myom}-gh-release.tar.xz
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# Installing ASAN version
echo "Installing ASAN version of STC OpenM++ for Ubuntu."
cp -fr "${HOME}/${myom}" "${HOME}/${myom}_ASAN"
cd "${HOME}/${myom}_ASAN/openm"
make clean-all
make USE_ASAN=1 libopenm
make USE_ASAN=1 RELEASE=1 libopenm
cd "${HOME}"
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# Update/Create .bashrc
cat "Updating .bashrc ..."
if [[ -f ".bashrc" ]];then
    cp -v "${HOME}/.bashrc" "${HOME}/.bashrc.bak"
    sed -i '/is the System-Wide OpenM++ Default Installation/Id' ".bashrc"
    sed -i '/ulimit -S -s 65536/Id' ".bashrc"
    sed -i '/export OMP_ROOT=/Id' ".bashrc"
    sed -i '/export OM_ROOT=/Id' ".bashrc"
    sed -i '/export OMA_ROOT=/Id' ".bashrc"
else
    touch ".bashrc"
fi

cat>>".bashrc"<<endmsg

# ${myom} is the System-Wide OpenM++ Default Installation
ulimit -S -s 65536
export OMP_ROOT=${HOME}/${myom}
export OM_ROOT=${HOME}/${myom}
export OMA_ROOT=${HOME}/${myom}_ASAN

endmsg

source ".bashrc"

############################################################################

echo ""
echo ""
echo "DONE!"
echo "${myom} is installed in ${HOME}."
echo "Environment variable 'OM_ROOT' was set to:"
echo ""
echo "    ${HOME}/${myom}"
echo ""
echo "in your '${HOME}/.bashrc' file."
echo "(It was created if it did not exist alread)."
read -p "Press the [ENTER] key to continue ..." < /dev/tty
