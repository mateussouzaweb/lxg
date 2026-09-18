#!/bin/sh

if [ -x "/usr/local/bin/lxg" ]; then
    exec /usr/local/bin/lxg container request xdg-open "$@"
else
    exec /usr/bin/xdg-open "$@"
fi