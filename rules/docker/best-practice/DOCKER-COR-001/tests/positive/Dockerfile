FROM ubuntu:22.04

# Bad: Multiple ENTRYPOINTs - only last one is used
ENTRYPOINT ["/bin/sh"]          # Ignored silently
ENTRYPOINT ["/usr/bin/python3"] # Ignored silently
ENTRYPOINT ["/app/start.sh"]    # Only this one takes effect
