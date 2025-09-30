#!/bin/bash

set -euo pipefail  # Exit on error, undefined vars, pipe failures

# Configuration
TARGETS_FILE="${TARGETS_FILE:-targets.txt}"
PLUGINS_DIR="/tmp/wordpress-plugins"
RESULTS_DIR="/app/results"
START_FROM="${1:-1}"  # Allow starting from a specific line number
COUNT="${2:-10}"      # Number of plugins to analyze (default: 10)
MAX_PLUGINS="${MAX_PLUGINS:-50}"  # Global limit for safety
SECUREFLOW_CMD="secureflow"

# DefectDojo Configuration
DEFECTDOJO_URL="http://127.0.0.1:8080"
DEFECTDOJO_TOKEN="3bd6b8cfcb8adf84974654a319afeea8647a9a18"
DEFECTDOJO_PRODUCT_ID="1"  # Default product ID

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Usage information
usage() {
    cat << EOF
Usage: $0 [START_FROM] [COUNT]

WordPress Plugin Security Analysis Tool

Arguments:
    START_FROM    Line number to start from in targets.txt (default: 1)
    COUNT         Number of plugins to analyze (default: 10)

Examples:
    $0              # Analyze first 10 plugins (lines 1-10)
    $0 5            # Analyze 10 plugins starting from line 5 (lines 5-14)
    $0 20 5         # Analyze 5 plugins starting from line 20 (lines 20-24)

Environment Variables:
    TARGETS_FILE    Path to targets file (default: targets.txt)
    MAX_PLUGINS     Global safety limit (default: 50)

EOF
}

# Validate parameters
validate_parameters() {
    # Check for help flag
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        usage
        exit 0
    fi
    
    # Validate START_FROM is a positive integer
    if ! [[ "$START_FROM" =~ ^[1-9][0-9]*$ ]]; then
        error "START_FROM must be a positive integer, got: $START_FROM"
        usage
        exit 1
    fi
    
    # Validate COUNT is a positive integer
    if ! [[ "$COUNT" =~ ^[1-9][0-9]*$ ]]; then
        error "COUNT must be a positive integer, got: $COUNT"
        usage
        exit 1
    fi
    
    # Enforce global safety limit
    if [[ $COUNT -gt $MAX_PLUGINS ]]; then
        error "COUNT ($COUNT) exceeds MAX_PLUGINS safety limit ($MAX_PLUGINS)"
        exit 1
    fi
    
    log "Parameters validated: START_FROM=$START_FROM, COUNT=$COUNT"
}

# Validate environment
validate_environment() {
    log "Validating environment..."
    
    # Check if running as root (security check)
    if [[ $EUID -eq 0 ]]; then
        error "This script should not be run as root for security reasons"
        exit 1
    fi
    
    # Check required commands
    for cmd in curl unzip secureflow jq; do
        if ! command -v "$cmd" &> /dev/null; then
            error "Required command '$cmd' not found"
            exit 1
        fi
    done
    
    # Check if targets file exists
    if [[ ! -f "$TARGETS_FILE" ]]; then
        error "Targets file '$TARGETS_FILE' not found"
        exit 1
    fi
    
    # Create directories
    mkdir -p "$PLUGINS_DIR" "$RESULTS_DIR"
    
    # Ensure results directory is writable
    if [[ ! -w "$RESULTS_DIR" ]]; then
        error "Results directory '$RESULTS_DIR' is not writable"
        exit 1
    fi
    
    success "Environment validation passed"
}

# Secure file permission setting
secure_files() {
    local plugin_dir="$1"
    log "Setting secure permissions for $plugin_dir"
    
    # Remove execute permissions and make files read-only
    find "$plugin_dir" -type f -exec chmod 444 {} \;
    # Make directories readable and traversable but not writable
    find "$plugin_dir" -type d -exec chmod 555 {} \;
    
    # Remove any potentially dangerous files
    find "$plugin_dir" -name "*.exe" -o -name "*.bat" -o -name "*.sh" -o -name "*.cmd" | while read -r file; do
        warn "Removing potentially dangerous file: $file"
        rm -f "$file"
    done
}

# Download and extract WordPress plugin
download_plugin() {
    local slug="$1"
    local plugin_dir="$PLUGINS_DIR/$slug"
    
    log "Downloading plugin: $slug"
    
    # Clean up any existing directory
    rm -rf "$plugin_dir"
    mkdir -p "$plugin_dir"
    
    # Download plugin zip
    local download_url="https://downloads.wordpress.org/plugin/${slug}.zip"
    local zip_file="/tmp/${slug}.zip"
    
    if curl -L -f -s -o "$zip_file" "$download_url"; then
        log "Downloaded $slug successfully"
        
        # Extract with size limits (prevent zip bombs)
        if unzip -q -o "$zip_file" -d "/tmp/extract_$$" 2>/dev/null; then
            # Handle WordPress plugin directory structure
            # Plugins are typically packaged as plugin-name/plugin-name/* 
            local extracted_dir="/tmp/extract_$$"
            local plugin_content_dir
            
            # Find the actual plugin directory (should be the only directory in extracted_dir)
            plugin_content_dir=$(find "$extracted_dir" -mindepth 1 -maxdepth 1 -type d | head -1)
            
            if [[ -n "$plugin_content_dir" && -d "$plugin_content_dir" ]]; then
                # Move the plugin content to our target directory
                mv "$plugin_content_dir"/* "$plugin_dir"/ 2>/dev/null || true
                # Handle hidden files if any
                mv "$plugin_content_dir"/.[^.]* "$plugin_dir"/ 2>/dev/null || true
                # Clean up temporary extraction directory
                rm -rf "$extracted_dir"
            else
                # Fallback: move everything from extraction directory
                mv "$extracted_dir"/* "$plugin_dir"/ 2>/dev/null || true
                mv "$extracted_dir"/.[^.]* "$plugin_dir"/ 2>/dev/null || true
                rm -rf "$extracted_dir"
            fi
            # Check extracted size (limit to 100MB)
            local size=$(du -sm "$plugin_dir" | cut -f1)
            if [[ $size -gt 100 ]]; then
                warn "Plugin $slug is too large (${size}MB), skipping"
                rm -rf "$plugin_dir" "$zip_file"
                return 1
            fi
            
            # Secure the files
            secure_files "$plugin_dir"
            
            # Clean up zip file
            rm -f "$zip_file"
            success "Plugin $slug extracted and secured"
            return 0
        else
            error "Failed to extract $slug"
            rm -rf "/tmp/extract_$$" "$zip_file"
            return 1
        fi
    else
        error "Failed to download $slug"
        return 1
    fi
}

# Run SecureFlow analysis
analyze_plugin() {
    local slug="$1"
    local plugin_dir="$PLUGINS_DIR/$slug"
    local date_stamp=$(date +%Y-%m-%d)
    local output_file="$RESULTS_DIR/${slug}-${date_stamp}-findings.json"
    
    log "Analyzing plugin: $slug"
    
    if [[ ! -d "$plugin_dir" ]]; then
        error "Plugin directory not found: $plugin_dir"
        return 1
    fi
    
    # Run SecureFlow CLI with DefectDojo output
    log "Running: $SECUREFLOW_CMD scan $plugin_dir --format defectdojo --defectdojo-url $DEFECTDOJO_URL --defectdojo-token $DEFECTDOJO_TOKEN --defectdojo-product-id $DEFECTDOJO_PRODUCT_ID --output $output_file"
    if "$SECUREFLOW_CMD" scan "$plugin_dir" --format defectdojo --defectdojo-url "$DEFECTDOJO_URL" --defectdojo-token "$DEFECTDOJO_TOKEN" --defectdojo-product-id "$DEFECTDOJO_PRODUCT_ID" --output "$output_file"; then
        if [[ -f "$output_file" && -s "$output_file" ]]; then
            # Validate JSON output
            if jq empty "$output_file" 2>/dev/null; then
                local findings_count=$(jq '.findings | length' "$output_file" 2>/dev/null || echo "0")
                success "Analysis complete for $slug: $findings_count findings"
                
                # Add metadata to the results file
                local temp_file=$(mktemp)
                jq --arg slug "$slug" --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg date_stamp "$date_stamp" \
                   '. + {plugin_slug: $slug, analysis_timestamp: $timestamp, analysis_date: $date_stamp}' \
                   "$output_file" > "$temp_file" && mv "$temp_file" "$output_file"
                
                return 0
            else
                error "Invalid JSON output for $slug"
                rm -f "$output_file"
                return 1
            fi
        else
            warn "No findings file generated for $slug"
            return 1
        fi
    else
        error "SecureFlow analysis failed for $slug"
        return 1
    fi
}

# Cleanup plugin directory after analysis
cleanup_plugin() {
    local slug="$1"
    local plugin_dir="$PLUGINS_DIR/$slug"
    
    if [[ -d "$plugin_dir" ]]; then
        # Change permissions back to allow deletion
        find "$plugin_dir" -type d -exec chmod 755 {} \;
        find "$plugin_dir" -type f -exec chmod 644 {} \;
        rm -rf "$plugin_dir"
        log "Cleaned up plugin directory: $slug"
    fi
}

# Main processing function
process_plugins() {
    local total_plugins=$(wc -l < "$TARGETS_FILE")
    local processed=0
    local successful=0
    local failed=0
    
    log "Starting plugin analysis from line $START_FROM, analyzing $COUNT plugins (total available: $total_plugins)"
    log "Will analyze lines $START_FROM to $((START_FROM + COUNT - 1))"
    
    # Read plugins starting from specified line, limited by COUNT
    tail -n +$START_FROM "$TARGETS_FILE" | head -n $COUNT | while IFS= read -r slug; do
        # Skip empty lines
        [[ -z "$slug" ]] && continue
        
        processed=$((processed + 1))
        local current_line=$((START_FROM + processed - 1))
        
        log "Processing plugin $processed/$COUNT (line $current_line): $slug"
        
        # Download plugin
        if download_plugin "$slug"; then
            # Analyze plugin
            if analyze_plugin "$slug"; then
                successful=$((successful + 1))
                success "Successfully analyzed $slug ($successful successful so far)"
            else
                failed=$((failed + 1))
                error "Failed to analyze $slug ($failed failed so far)"
            fi
            
            # Always cleanup after analysis
            cleanup_plugin "$slug"
        else
            failed=$((failed + 1))
            error "Failed to download $slug ($failed failed so far)"
        fi
        
        # Progress update
        log "Progress: $processed processed, $successful successful, $failed failed"
        
        # Small delay to be respectful to WordPress.org
        sleep 2
    done
    
    log "Analysis complete! Processed: $processed, Successful: $successful, Failed: $failed"
    log "Results saved in: $RESULTS_DIR"
}

# Signal handlers for graceful shutdown
cleanup_on_exit() {
    log "Received interrupt signal, cleaning up..."
    # Kill any background processes
    jobs -p | xargs -r kill
    exit 130
}

trap cleanup_on_exit INT TERM

# Main execution
main() {
    log "Starting WordPress plugin security analysis"
    
    validate_parameters "$@"
    
    log "Configuration:"
    log "  Start from line: $START_FROM"
    log "  Count to analyze: $COUNT"
    log "  Max plugins limit: $MAX_PLUGINS"
    log "  Results directory: $RESULTS_DIR"
    log "  Targets file: $TARGETS_FILE"
    
    validate_environment
    process_plugins
    
    log "Plugin analysis completed successfully"
}

# Run main function
main "$@"
