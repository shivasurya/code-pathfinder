"""
DOCKER-AUD-001: Dockerfile Source Not Pinned

Audit: Image digest pinning

DESCRIPTION:
Detects FROM instructions without digest pinning (@sha256:...).
Digest pinning ensures exact image reproducibility.

EXAMPLE:
```dockerfile
# Not pinned - tag can change
FROM nginx:1.24.0

# Pinned with digest - immutable
FROM nginx:1.24.0@sha256:abcd1234...
```

REMEDIATION:
Pin images with digests for critical production builds.
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-AUD-001",
    name="Dockerfile Source Not Pinned",
    severity="LOW",
    category="audit",
    message="FROM instruction without digest pinning. Consider using @sha256:... for immutable builds."
)
def dockerfile_source_not_pinned():
    return instruction(type="FROM", missing_digest=True)
