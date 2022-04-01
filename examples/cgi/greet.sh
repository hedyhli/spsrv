#!/usr/bin/env sh

# User can enter their name and this will greet them
# Example inpu link:
#
# =: greet.sh Enter your name

printf "2 text/plain\r\n"
name=$(cat /dev/stdin)
echo "Hello, ${name:-World}!"
