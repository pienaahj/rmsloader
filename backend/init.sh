#!/bin/sh

# echo "Hello from init.sh"

# Set the directory where the script resides
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

echo "INITIALIZATION_MODE: $INITIALIZATION_MODE"

# Check if INITIALIZATION_MODE is set to "p"
if [ "$INITIALIZATION_MODE" = "p" ]; then
    # Run initialization tasks
    echo "Running initialization tasks..."
    # Create the logs directory
    if [ ! -d "$SCRIPT_DIR/logs" ]; then
        echo "Creating logs directory..."
        mkdir -p "$SCRIPT_DIR/logs"
        chmod -R 777 "$SCRIPT_DIR/logs"
        # ls -la "$SCRIPT_DIR"
    fi
    # Start the application in process mode
    ./run -mode=p
else
    # Run normal startup tasks
    echo "Running normal startup tasks..."
    # Start the application in server mode
    ./run -mode=s
fi

