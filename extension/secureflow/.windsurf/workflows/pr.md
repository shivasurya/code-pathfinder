---
description: Pull request creation workflow
auto_execution_mode: 1
---

This workflow is dedicated for creating github pull request from command line.

Follow the instruction below to create a descriptive pull request based on commits comparing to main

1. Ensure you're in a branch apart from main / default
2. Go through the commits in the branch comparing to main (or default branch)
3. Go through the commit message, code changes
4. Come up with good decent pull request message body
5. use gh label list command to learn about available labels
6. Always choose the perfect label for the pull request
7. Assign the pull request to me `shivasurya` always
8. Follow pr title format as secureflow/{feature/release/chore/bugfix/enhancement/RELEVANT one}: contextual title
    for example: secureflow/bugfix: Add sentry filtering to reduce noise for reported issues 
9. Using gh pr create command fillup the details as per the above collected context
10. finally give me the pr link