#!/bin/bash

if ! command -v "pgrep" > /dev/null
then
    echo "pgrep could not be found on PATH, but is necessary for running jim. "
    exit
fi

if ! pgrep -x "jimServer" > /dev/null
then
    ./jimServer &
fi

./jimClient "$@"
