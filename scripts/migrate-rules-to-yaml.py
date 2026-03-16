#!/usr/bin/env python3
"""
Migrate legacy single-file rules to the YAML folder structure.

For each rule file:
  old: rules/{lang}/{category}/{rule_name}.py  (docstring + decorator + function)
  new: rules/{lang}/{category}/{RULE-ID}/
       ├── rule.py      (imports + decorator + function, no docstring)
       ├── meta.yaml    (metadata extracted from decorator + docstring)
       └── tests/
           ├── positive/  (VULNERABLE EXAMPLE from docstring)
           └── negative/  (SECURE EXAMPLE from docstring)

Usage:
  python scripts/migrate-rules-to-yaml.py rules/docker/security/secret_in_build_arg.py
  python scripts/migrate-rules-to-yaml.py rules/docker/  # all files in dir
  python scripts/migrate-rules-to-yaml.py --dry-run rules/docker/
"""

import os
import re
import sys
import yaml
import textwrap

DECORATOR_PATTERN = re.compile(
    r'@(dockerfile_rule|compose_rule|python_rule|java_rule)\s*\(([\s\S]*?)\)\s*\ndef\s+(\w+)',
    re.MULTILINE
)

def parse_decorator_params(content):
    """Parse key=value pairs from decorator content."""
    params = {}
    # Match key="value" or key='value'
    for match in re.finditer(r'(\w+)\s*=\s*("(?:[^"\\]|\\.)*"|\'(?:[^\'\\]|\\.)*\')', content):
        key = match.group(1)
        value = match.group(2)[1:-1]  # strip quotes
        params[key] = value
    return params

def extract_docstring_section(docstring, section_name):
    """Extract a section from the docstring by name."""
    pattern = rf'{section_name}:\s*\n```(?:python|dockerfile|bash|yaml)?\n([\s\S]*?)```'
    match = re.search(pattern, docstring, re.IGNORECASE)
    if match:
        return match.group(1).strip()
    return ''

def extract_full_section(docstring, section_name):
    """Extract full text of a section until the next section header."""
    # Find section start
    pattern = rf'^{section_name}:\s*\n([\s\S]*?)(?=\n[A-Z][A-Z ]+:\s*\n|\Z)'
    match = re.search(pattern, docstring, re.MULTILINE | re.IGNORECASE)
    if match:
        return match.group(1).strip()
    return ''

def extract_security_implications(docstring):
    """Parse numbered security implications from docstring."""
    section = extract_full_section(docstring, 'SECURITY IMPLICATIONS')
    if not section:
        return []

    implications = []
    # Match patterns like "1. Title: description" or "**1. Title**:"
    parts = re.split(r'\n\d+\.\s+\*?\*?', section)
    for part in parts[1:]:  # skip preamble before first number
        part = part.strip()
        if not part:
            continue
        # Try to extract title from "Title: rest" or "**Title**: rest"
        title_match = re.match(r'\*?\*?([^:*]+)\*?\*?:\s*([\s\S]*)', part)
        if title_match:
            title = title_match.group(1).strip()
            desc = title_match.group(2).strip()
            # Clean up multiline descriptions
            desc = re.sub(r'\n\s+', ' ', desc)
            implications.append({'title': title, 'description': desc})

    return implications

def get_cwe_name(cwe_id):
    """Map common CWE IDs to names."""
    cwe_names = {
        'CWE-250': 'Execution with Unnecessary Privileges',
        'CWE-538': 'Insertion of Sensitive Information into Externally-Accessible File or Directory',
        'CWE-200': 'Exposure of Sensitive Information to an Unauthorized Actor',
        'CWE-264': 'Permissions, Privileges, and Access Controls',
        'CWE-732': 'Incorrect Permission Assignment for Critical Resource',
        'CWE-770': 'Allocation of Resources Without Limits or Throttling',
        'CWE-863': 'Incorrect Authorization',
        'CWE-269': 'Improper Privilege Management',
        'CWE-284': 'Improper Access Control',
        'CWE-668': 'Exposure of Resource to Wrong Sphere',
        'CWE-399': 'Resource Management Errors',
        'CWE-676': 'Use of Potentially Dangerous Function',
        'CWE-829': 'Inclusion of Functionality from Untrusted Control Sphere',
        'CWE-1078': 'Inappropriate Source Code Style or Formatting',
        'CWE-561': 'Dead Code',
        'CWE-710': 'Improper Adherence to Coding Standards',
        'CWE-489': 'Active Debug Code',
    }
    return cwe_names.get(cwe_id, cwe_id.replace('CWE-', 'CWE '))

def migrate_rule(filepath, dry_run=False):
    """Migrate a single rule file to the YAML folder structure."""
    content = open(filepath).read()

    # Find all decorators in the file
    matches = list(DECORATOR_PATTERN.finditer(content))
    if not matches:
        print(f'  SKIP {filepath} (no rule decorator found)')
        return 0

    if len(matches) > 1:
        print(f'  SKIP {filepath} ({len(matches)} rules -- split manually)')
        return 0

    match = matches[0]
    decorator_type = match.group(1)
    decorator_content = match.group(2)
    func_name = match.group(3)
    params = parse_decorator_params(decorator_content)

    rule_id = params.get('id', '')
    if not rule_id:
        print(f'  SKIP {filepath} (no rule id)')
        return 0

    # Extract module docstring
    docstring_match = re.match(r'^"""([\s\S]*?)"""', content)
    docstring = docstring_match.group(1).strip() if docstring_match else ''

    # Extract examples from docstring
    vulnerable_code = extract_docstring_section(docstring, 'VULNERABLE EXAMPLE')
    secure_code = extract_docstring_section(docstring, 'SECURE EXAMPLE')
    if not secure_code:
        secure_code = extract_docstring_section(docstring, 'SECURE ALTERNATIVES')

    # Extract description
    description = extract_full_section(docstring, 'DESCRIPTION')

    # Extract security implications
    implications = extract_security_implications(docstring)

    # Extract references
    refs_text = extract_full_section(docstring, 'REFERENCES')
    references = []
    for line in refs_text.split('\n'):
        line = line.strip().lstrip('- ')
        if line:
            references.append({'title': line, 'url': ''})

    # Determine language and category path
    dirpath = os.path.dirname(filepath)
    category = params.get('category', os.path.basename(dirpath))

    # Determine language from decorator type
    if decorator_type == 'dockerfile_rule':
        language = 'docker'
    elif decorator_type == 'compose_rule':
        language = 'docker-compose'
    elif decorator_type == 'python_rule':
        language = 'python'
    else:
        language = 'unknown'

    # Build the rule code (everything from imports to end, minus docstring)
    if docstring_match:
        code_start = docstring_match.end()
        rule_code = content[code_start:].strip()
    else:
        rule_code = content.strip()

    # Build CWE info
    cwe_id = params.get('cwe', '')
    cwe_list = []
    if cwe_id:
        cwe_list.append({
            'id': cwe_id,
            'name': get_cwe_name(cwe_id),
            'url': f'https://cwe.mitre.org/data/definitions/{cwe_id.replace("CWE-", "")}.html'
        })

    # Build tags
    tags_str = params.get('tags', '')
    tags = [t.strip() for t in tags_str.split(',') if t.strip()]

    # Build meta.yaml
    meta = {
        'id': rule_id,
        'name': params.get('name', func_name),
        'short_description': params.get('message', ''),
        'severity': params.get('severity', 'MEDIUM').upper(),
        'category': category,
        'language': language,
        'ruleset': f'{language}/{rule_id}',
        'cwe': cwe_list,
        'tags': tags,
        'author': {'name': 'Shivasurya', 'url': 'https://x.com/sshivasurya'},
        'description': description or params.get('message', ''),
        'message': params.get('message', ''),
        'secure_example': secure_code,
        'recommendations': [],
        'detection_scope': '',
        'references': references,
        'compliance': [],
        'faq': [],
        'similar_rules': [],
        'tests': {
            'positive': 'tests/positive/',
            'negative': 'tests/negative/',
        },
    }

    if implications:
        meta['security_implications'] = implications

    # Determine output folder
    out_dir = os.path.join(dirpath, rule_id)

    if dry_run:
        print(f'  DRY  {rule_id} -> {out_dir}/')
        return 1

    # Create folder structure
    os.makedirs(os.path.join(out_dir, 'tests', 'positive'), exist_ok=True)
    os.makedirs(os.path.join(out_dir, 'tests', 'negative'), exist_ok=True)

    # Write rule.py
    with open(os.path.join(out_dir, 'rule.py'), 'w') as f:
        f.write(rule_code + '\n')

    # Write meta.yaml
    with open(os.path.join(out_dir, 'meta.yaml'), 'w') as f:
        yaml.dump(meta, f, default_flow_style=False, allow_unicode=True, sort_keys=False, width=100)

    # Write vulnerable example as test
    if vulnerable_code:
        # Determine file extension based on language
        if language in ('docker', 'docker-compose'):
            test_file = 'Dockerfile' if language == 'docker' else 'docker-compose.yml'
        else:
            test_file = 'vulnerable.py'
        with open(os.path.join(out_dir, 'tests', 'positive', test_file), 'w') as f:
            f.write(vulnerable_code + '\n')

    # Write secure example as negative test
    if secure_code:
        if language in ('docker', 'docker-compose'):
            test_file = 'Dockerfile' if language == 'docker' else 'docker-compose.yml'
        else:
            test_file = 'safe.py'
        with open(os.path.join(out_dir, 'tests', 'negative', test_file), 'w') as f:
            f.write(secure_code + '\n')

    print(f'  OK   {rule_id} -> {out_dir}/')
    return 1


def main():
    dry_run = '--dry-run' in sys.argv
    paths = [a for a in sys.argv[1:] if a != '--dry-run']

    if not paths:
        print('Usage: migrate-rules-to-yaml.py [--dry-run] <file_or_dir> ...')
        sys.exit(1)

    total = 0
    for path in paths:
        if os.path.isfile(path):
            total += migrate_rule(path, dry_run)
        elif os.path.isdir(path):
            for root, dirs, files in os.walk(path):
                # Skip already-migrated YAML folders
                if 'meta.yaml' in files:
                    dirs.clear()
                    continue
                for f in sorted(files):
                    if f.endswith('.py') and not f.startswith('__'):
                        total += migrate_rule(os.path.join(root, f), dry_run)

    print(f'\n{"Would migrate" if dry_run else "Migrated"}: {total} rules')


if __name__ == '__main__':
    main()
