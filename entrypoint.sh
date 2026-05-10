#!/bin/sh
set -e

# Generate home page on startup
./hndigest home

# Run cron job every hour
echo "0 * * * * cd /app && ./hndigest home" | crontab -

# Start cron and keep container alive
crond -f -l 2
