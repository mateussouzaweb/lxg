#!/bin/sh

if [ -x "/usr/local/bin/lxg" ]; then
    eval "$(/usr/local/bin/lxg container init 2>/dev/null)"
fi