#!/bin/bash

# The file where we'll save our target plugin slugs
TARGET_FILE="targets.txt" 
echo "" > "$TARGET_FILE" # Clear the file to start fresh

# Let's check pages of the "popular" plugins list.
# Each page has 100 plugins, so this will search through plugins with lower install counts.
for page in {25..30}; do
    echo "Fetching page $page..."
    
    # Query the API and filter the results with jq
    # Note: URL-encoded square brackets and adjusted install count criteria
    slugs=$(curl -s "https://api.wordpress.org/plugins/info/1.2/?action=query_plugins&request%5Bbrowse%5D=popular&request%5Bpage%5D=$page&request%5Bper_page%5D=100" | \
            jq -r '.plugins[] | select(.active_installs >= 1000 and .active_installs <= 20000) | .slug')

    # If we found any matching slugs, add them to our target file
    if [[ -n "$slugs" ]]; then
        echo "$slugs" >> "$TARGET_FILE"
        echo "Found $(echo "$slugs" | wc -l) plugins on page $page"
    else
        echo "No matching plugins found on page $page"
    fi
done

echo "Done. Target list saved to $TARGET_FILE"
echo "$(wc -l < "$TARGET_FILE") plugins found matching your criteria."