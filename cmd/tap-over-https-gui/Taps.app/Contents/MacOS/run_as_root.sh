#!/usr/bin/env bash
taps=$(dirname $0)/Taps
/usr/bin/osascript <<EOF
do shell script "$taps" with prompt "启动Taps需要授权" with administrator privileges
EOF
