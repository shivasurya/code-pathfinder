"""
DOCKER-BP-017: Use WORKDIR Instead of cd

Best Practice: Use WORKDIR for changing directories

DESCRIPTION:
Using 'cd' in RUN commands is error-prone and doesn't persist.
WORKDIR is explicit and affects all subsequent instructions.

EXAMPLE:
```dockerfile
# Bad - cd doesn't persist
RUN cd /app && npm install

# Good - WORKDIR persists
WORKDIR /app
RUN npm install
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction, missing
from rules.container_combinators import all_of, any_of


@dockerfile_rule(
    id="DOCKER-BP-017",
    name="Use WORKDIR Instead of cd",
    severity="LOW",
    category="best-practice",
    message="Use WORKDIR instruction instead of 'cd' in RUN commands."
)
def use_workdir():
    return all_of(
        any_of(
            instruction(type="RUN", contains=" cd "),
            instruction(type="RUN", regex=r"\bcd\s+")
        ),
        missing(instruction="WORKDIR")
    )
