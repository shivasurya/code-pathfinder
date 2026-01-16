"""
DOCKER-BP-011: Prefer COPY Over ADD

Best Practice: Use COPY for simple file operations

DESCRIPTION:
Detects use of ADD instruction when COPY would suffice. ADD has implicit behavior
(auto-extraction of tar archives, URL downloading) that can be surprising and
create security risks.

WHY COPY IS BETTER:
- More transparent and predictable
- Doesn't auto-extract archives unexpectedly
- Doesn't perform network operations
- Better build cache utilization

VULNERABLE EXAMPLE:
```dockerfile
ADD app.tar.gz /app/  # ❌ Auto-extracts (might be unintended)
ADD https://example.com/file /tmp/  # ❌ Downloads from network
```

SECURE EXAMPLE:
```dockerfile
COPY app.tar.gz /app/  # ✅ Just copies the file
RUN tar xzf /app/app.tar.gz  # ✅ Explicit extraction
```

WHEN TO USE ADD:
- Only when you specifically need auto-extraction
- Document why ADD is needed

REMEDIATION:
Replace ADD with COPY unless you specifically need ADD's special features.

REFERENCES:
- Docker Best Practices
- hadolint DL3020
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-011",
    name="Prefer COPY Over ADD",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,copy,add,file-operations,best-practice,transparency,predictability,anti-pattern",
    message="Use COPY instead of ADD for simple file operations. ADD has implicit behavior that can be surprising."
)
def prefer_copy_over_add():
    return instruction(type="ADD")
