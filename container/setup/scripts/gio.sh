#!/bin/sh

if [ -n "$LXG_CONTAINER" ] && \
   [ -x "/usr/local/bin/lxg" ] && \
   [ "$1" = "open" -o "$1" = "launch" ]; }; then
    exec /usr/local/bin/lxg container request gio "$@"
else
    exec /usr/bin/gio "$@"
fi