#!/bin/bash
PID=$1
OUTPUT=${2:-memory_usage.csv}

echo "timestamp,rss_mb,vsz_mb" > $OUTPUT

while kill -0 $PID 2>/dev/null; do
    TIMESTAMP=$(date +%s.%N)
    MEM=$(ps -p $PID -o rss=,vsz= 2>/dev/null | awk '{print $1/1024","$2/1024}')
    if [ ! -z "$MEM" ]; then
        echo "$TIMESTAMP,$MEM" >> $OUTPUT
    fi
    sleep 0.1  # Sample every 100ms instead of 500ms
done

echo "Memory monitoring complete. Data saved to $OUTPUT"
