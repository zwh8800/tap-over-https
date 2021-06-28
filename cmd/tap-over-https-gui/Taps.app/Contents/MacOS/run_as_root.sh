#!/usr/bin/env bash
taps=$(dirname $0)/Taps
/usr/bin/osascript <<EOF
do shell script "$taps > /dev/null 2>&1 &" with prompt "启动Taps需要授权" with administrator privileges
EOF
