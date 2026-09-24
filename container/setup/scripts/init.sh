#!/bin/sh

if [ -x "/usr/local/bin/lxg" ]; then
    export LXG_CONTAINER="1"
    eval "$(/usr/local/bin/lxg container init 2>/dev/null)"
fi