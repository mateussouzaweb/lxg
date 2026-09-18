#!/bin/sh

if [ -x "/usr/local/bin/lxg" ] && [ "$1" = "open" ]; then
    exec /usr/local/bin/lxg bridge exec gio "$@"
else
    exec /usr/bin/gio "$@"
fi