#!/bin/bash

# Purpose: Wrapper for building local binaries without updating user facing prod/dev releases
# Requires: MacOS or Linux OS with GoLang 1.13 or higher
# Usage: ./build_local_binaries.sh <version> <stage> <directory> <filename>
#                                  version - (default "0.0.0")
#                                  stage - Options: [dev prod] (default "dev")
#                                  directory - (Optional)
#                                  filename - (Optional)
# Examples: 
#     ./build_local_binaries.sh 5.0 development                                 (to build locsl development binaries without installing)
#     ./build_local_binaries.sh 5.0 development /usr/local/bin aws-okta-test    (to build and install locsl development binaries for testing)

if [ -z ${1} ]; then
    VERSION="0.0.0"
else
    VERSION=$1
fi

if [ -z ${2} ]; then
    STAGE="dev"
else
    STAGE=$2
fi

if [ -z ${3} ]; then
    SCOPE="BUILD-ONLY"
else
    DIR=$3
    if [ -z ${4} ]; then
        FILENAME="aws-okta-local"
    else
        FILENAME=$4
    fi
    SCOPE="BUILD-AND-INSTALL"
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

echo "Building local $STAGE binaries for aws-okta $VERSION"
make "$STAGE" -e VERSION="$VERSION" RELEASE="NORDSTROM AWS-OKTA CLI - $STAGE - COMPILED ($(date +'%m/%d/%Y'))"
if [ $? != 0 ]; then
    exit $?
fi

if [ $SCOPE == "BUILD-AND-INSTALL" ]; then
    echo "Copying $OS binary to $DIR/$FILENAME"
    cp release/*$OS* $DIR/$FILENAME
    if [ $? != 0 ]; then
        exit $?
    fi

    chmod +x $DIR/$FILENAME
    if [ $? != 0 ]; then
        exit $?
    fi
fi