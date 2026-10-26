#!/bin/bash
# Install dependencies
echo "INSTALLING OpenM++ DEPENDENCIES ..."
sudo apt install -y gcc-12
sudo apt install -y g++-12
sudo update-alternatives --remove-all gcc 
sudo update-alternatives --remove-all g++
sudo update-alternatives --install /usr/bin/gcc gcc /usr/bin/gcc-11 10
sudo update-alternatives --install /usr/bin/gcc gcc /usr/bin/gcc-12 20
sudo update-alternatives --install /usr/bin/g++ g++ /usr/bin/g++-11 10
sudo update-alternatives --install /usr/bin/g++ g++ /usr/bin/g++-12 20
sudo update-alternatives --install /usr/bin/cc cc /usr/bin/gcc 30
sudo update-alternatives --set cc /usr/bin/gcc
sudo update-alternatives --install /usr/bin/c++ c++ /usr/bin/g++ 30
sudo update-alternatives --set c++ /usr/bin/g++
sudo apt install -y sqlite
sudo apt install -y gdb
sudo apt install -y lynx
sudo apt install -y flex
sudo apt install -y bison
sudo apt install -y make

echo '############################################################################'
