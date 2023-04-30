#!/bin/bash
set +e
# Set the name of the Golang binary file
binary_name="Main"

# Capture the output of the pgrep command into a variable
pid=$(pgrep $binary_name)

# Check if the pid variable is not empty (i.e., the program is running)
if [[ -n $pid ]]; then
    echo "$binary_name is running with PID $pid"
    exit 0
    # Run some other command or script here
else
    echo "$binary_name is not running"
    # Start the program here
    cd /var/go/chatgpt && ./$binary_name
    exit 0
fi
