#!/bin/sh
set -e

# Generate both home and daily pages on startup
./hndigest home
./hndigest daily

# Run cron job every 30 minutes
echo "*/30 * * * * cd /app && ./hndigest home && ./hndigest daily" | crontab -

# Start cron and keep container alive
crond -f -l 2
