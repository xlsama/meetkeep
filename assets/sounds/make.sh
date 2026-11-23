#!/bin/sh
set -e
cd "$(dirname "$0")"
make() { name=$1; shift; earshot make --mono -o "$name.wav" "$@"; }
make start --sound toggle/rocker_on --lufs -29
make stop --sound toggle/rocker_off --lufs -29
make pause --sound toggle/key_down --lufs -30
make resume --sound toggle/key_up --lufs -30
make notice --sound notify/lift --voice felt_piano --dur 1.3 --lufs -20
make ready --sound success/done --voice felt_piano --dur 1.3 --lufs -20
make fault --sound warning/pulse --voice felt_piano --dur 1.5 --lufs -18
