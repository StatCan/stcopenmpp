#!/bin/bash
# Get version to be installed
echo "GETTING ALL AVAILABLE VERSIONS OF OpenM++ ..."
cd "${HOME}"

sudo apt install -y lynx
sudo apt install -y dos2unix

lynx -dump https://github.com/statcan/stcopenmpp/tags > tmp.txt
cat>get_version.vim<<'endmsg'
:v/\/tags\//d
:v/\.tar\.gz/d
:%s/^.*tags\/\(v.*\)\.tar\.gz$/\1/g
:sort! u
:wq
endmsg
vim -T dumb --noplugin -n -es -S "${HOME}/get_version.vim" "${HOME}/tmp.txt"

readarray -t myv < "${HOME}/tmp.txt"
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
rm -f "/${HOME}/get_version.vim" "${HOME}/tmp.txt"

# get version .tar.gz file
uv=$(lsb_release -r)
uv=$(echo ${uv} | sed -e 's/^Release:[[:space:]]*\([0-9][0-9]\.[0-9][0-9].*\)$/\1/g')

echo "Installing STC OpenM++ for Ubuntu ${uv}"
if [[ ${uv} == "22.04" ]]; then
    myom=stcopenmpp-ubuntu22-${myv}
else
    myom=stcopenmpp-ubuntu-${myv}
fi
myurl=https://github.com/StatCan/stcopenmpp/releases/download/${myv}/${myom}-gh-release.tar.xz

echo "URL of version to be installed: ${myurl}"
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# Dowload
cat "Downloading ${myom} ..."
cd "/opt"
wget ${myurl}

# Extract
cat "Extracting ${myom} ..."
tar -xvf ${myom}-gh-release.tar.xz
rm -fr ${myom}-gh-release.tar.xz
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# Installing ASAN version
echo "Installing ASAN version of STC OpenM++ for Ubuntu."
sudo cp -vfr "/opt/${myom}" "/opt/${myom}_ASAN"
cd "/opt/${myom}_ASAN"
sudo make clean-all
sudo make USE_ASAN=1 libopenm
sudo make USE_ASAN=1 RELEASE=1 libopenm
cd "/opt"
sudo chmod -R 777 "/opt/${myom}_ASAN"
read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# Update /etc/bash.bashrc
cat "Updating bash.bashrc ..."
sudo cp -vf "/etc/bash.bashrc" "${HOME}/bash.bashrc"
sudo chown ${USER}:appadmins "${HOME}/bash.bashrc"
cp -vf "${HOME}/bash.bashrc" "${HOME}/bash.bashrc.bak"

sed -i '/is the System-Wide OpenM++ Default Installation/Id' "${HOME}/bash.bashrc"
sed -i '/ulimit -S -s 65536/Id' "${HOME}/bash.bashrc"
sed -i '/export OMP_ROOT=/Id' "${HOME}/bash.bashrc"
sed -i '/export OM_ROOT=/Id' "${HOME}/bash.bashrc"
sed -i '/export OMA_ROOT=/Id' "${HOME}/bash.bashrc"
sed -i '/Set the LD_LIBRARY_PATH/Id' "${HOME}/bash.bashrc"
sed -i '/LD_LIBRARY_PATH=/Id' "${HOME}/bash.bashrc"

cat>>"${HOME}/bash.bashrc"<<endmsg

# ${myom} is the System-Wide OpenM++ Default Installation
ulimit -S -s 65536
export OMP_ROOT=/opt/${myom}
export OM_ROOT=/opt/${myom}
export OMA_ROOT=/opt/${myom}_ASAN

endmsg

sudo chown root:root "${HOME}/bash.bashrc"
sudo mv -vf "${HOME}/bash.bashrc" "/etc/bash.bashrc"

source "/etc/bash.bashrc"

read -p "Press the [ENTER] key to continue ..." < /dev/tty

############################################################################

# Update .bashrc
cat "Updating ${HOME}/.bashrc ..."
if [[ -f ".bashrc" ]];then
    cp -v "${HOME}/.bashrc" "${HOME}/.bashrc.bak"
    sed -i '/is the System-Wide OpenM++ Default Installation/Id' "${HOME}/.bashrc"
    sed -i '/ulimit -S -s 65536/Id' "${HOME}/.bashrc"
    sed -i '/export OMP_ROOT=/Id' "${HOME}/.bashrc"
    sed -i '/export OM_ROOT=/Id' "${HOME}/.bashrc"
    sed -i '/export OMA_ROOT=/Id' "${HOME}/.bashrc"
    sed -i '/Set the LD_LIBRARY_PATH/Id' "${HOME}/.bashrc"
    sed -i '/LD_LIBRARY_PATH=/Id' "${HOME}/.bashrc"
else
    touch "${HOME}/.bashrc"
fi

cat>>"${HOME}/.bashrc"<<endmsg

# ${myom} is the System-Wide OpenM++ Default Installation
ulimit -S -s 65536
export OMP_ROOT=/opt/${myom}
export OM_ROOT=/opt/${myom}
export OMA_ROOT=

endmsg

source "${HOME}/.bashrc"

############################################################################

echo "DONE!"
echo "${myom} is installed in ${HOME}."
echo "Environment variable 'OM_ROOT' was set to:"
echo ""
echo "    /opt/${myom}"
echo ""
echo "in the system '/etc/bash.bashrc' file."
echo "and in your '${HOME}/.bashrc' file."
echo "(It was created if it did not exist alread)."
read -p "Press the [ENTER] key to continue ..." < /dev/tty
