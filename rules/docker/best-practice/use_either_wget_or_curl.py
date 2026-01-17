"""
DOCKER-BP-024: Install Only One of wget or curl

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects installation of both wget and curl in the same Dockerfile.
Both tools serve the same purpose (downloading files), so installing both
wastes image space. Choose one tool and use it consistently.

WHY THIS IS PROBLEMATIC:
1. Wasted Space: Installing both adds unnecessary MBs to image
2. Redundancy: Both tools have overlapping functionality
3. Confusion: Unclear which tool to use in the image
4. Maintenance: Two tools to update instead of one

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: Installing both wget and curl wastes space
RUN apt-get update && apt-get install -y wget curl
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Choose one tool (curl is more feature-rich)
RUN apt-get update && apt-get install -y curl

# Or use wget if you prefer
# RUN apt-get update && apt-get install -y wget
```

REMEDIATION:
Choose either wget or curl based on your needs:
- curl: More feature-rich, supports more protocols (HTTP, HTTPS, FTP, SFTP, etc.)
- wget: Simpler, better for recursive downloads

Remove the other tool from your package installation commands.

REFERENCES:
- Docker Best Practices
- hadolint DL4001
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-024",
    name="Install Only One of wget or curl",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,wget,curl,download,tools,optimization,image-size,redundancy,best-practice",
    message="Installing both wget and curl wastes space. Choose one tool for downloads."
)
def use_either_wget_or_curl():
    return all_of(
        instruction(type="RUN", contains="wget"),
        instruction(type="RUN", contains="curl")
    )
