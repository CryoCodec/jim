#!/bin/bash

if ! command -v "pgrep" > /dev/null
then
    echo "It seems a necessary utility is missing, running doctor command"
    ./jimClient doctor
    exit
fi

if ! pgrep -x "jimServer" > /dev/null
then
    ./jimServer &
fi

./jimClient "$@"
