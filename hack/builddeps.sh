#!/usr/bin/env bash
os=$(uname)
sudo=''
if [ "$os" = 'Darwin' ]; then
    brew install libyaml
elif [ "$os" = 'Linux' ]; then
    if [ $(lsb_release -is) = "Debian" ]; then
        apt-get install -y python-dev libyaml-dev python-pip build-essential libsqlite3-dev git curl
    else
        sudo apt-get -y install libyaml-dev build-essential libsqlite3-dev git curl
        sudo='sudo'
    fi
fi

# https://github.com/ellisonbg/antipackage
pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage
go get -u golang.org/x/tools/cmd/goimports
go get -u github.com/sgotti/glide-vc
curl https://glide.sh/get | sh