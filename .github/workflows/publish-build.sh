#!/bin/bash -e
if [ "${os}" == "" ]; then
    1>&2 echo "os unset"
    exit 1
fi
if [ "${arch}" == "" ]; then
    1>&2 echo "arch unset"
    exit 1
fi
if [ "${version}" == "" ]; then
    1>&2 echo "version unset"
    exit 1
fi

if [ "${os}" == "darwin" ]; then
    echo "install dependencies"
    brew install \
        openssl \
        readline \
        sqlite3 \
        xz \
        zlib \
        tcl-tk

    echo "setting macos python configure opts"
    export PYTHON_CONFIGURE_OPTS="--enable-framework"
    echo "PYTHON_CONFIGURE_OPTS=${PYTHON_CONFIGURE_OPTS}"
elif [ "${os}" == "linux" ]; then
    echo "install dependencies"
    apt -y update
    apt -y install \
        git \
        curl \
        build-essential \
        zlib1g-dev \
        libncurses5-dev \
        libgdbm-dev \
        liblzma-dev \
        libnss3-dev \
        libssl-dev \
        libreadline-dev \
        libffi-dev \
        libsqlite3-dev wget \
        libbz2-dev \
        patchelf
else
    1>&2 echo "not implemented: ${os}"
    exit 1
fi

echo "configure git safe.directory"
git config --global --add safe.directory "$(pwd)"

echo "install asdf"
git clone https://github.com/asdf-vm/asdf.git ${HOME}/.asdf --branch v0.14.0
. "${HOME}/.asdf/asdf.sh"

echo "install python"
asdf plugin add python
asdf install python 3.11.6
asdf shell python 3.11.6

echo "create virtualenv"
python -m venv /tmp/venv
. /tmp/venv/bin/activate

echo "install dependencies"
python -m pip install -e ".[dev]"

echo "setting version ${version}"
versionctl set "${version}" pyproject.toml

echo "building"
./scripts/build.py --output "build/versionctl-${os}-${arch}"