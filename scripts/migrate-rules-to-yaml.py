#!/usr/bin/env python3
"""
Migrate legacy single-file rules to the YAML folder structure.

Handles both single-rule and multi-rule files.

For each @python_rule/@dockerfile_rule/@compose_rule in a file:
  new: rules/{lang}/{category}/{RULE-ID}/
       ├── rule.py      (imports + relevant classes + decorator + function)
       ├── meta.yaml    (metadata from decorator + docstring)
       └── tests/
           ├── positive/  (VULNERABLE EXAMPLE from docstring)
           └── negative/  (SECURE EXAMPLE from docstring)

Usage:
  python scripts/migrate-rules-to-yaml.py rules/docker/
  python scripts/migrate-rules-to-yaml.py rules/python/lang/
  python scripts/migrate-rules-to-yaml.py --dry-run rules/python/
"""

import os
import re
import sys
import yaml

DECORATOR_PATTERN = re.compile(
    r'@(dockerfile_rule|compose_rule|python_rule|java_rule)\s*\(([\s\S]*?)\)\s*\ndef\s+(\w+)',
    re.MULTILINE
)


def parse_decorator_params(content):
    params = {}
    for match in re.finditer(r'(\w+)\s*=\s*("(?:[^"\\]|\\.)*"|\'(?:[^\'\\]|\\.)*\')', content):
        key = match.group(1)
        value = match.group(2)[1:-1]
        params[key] = value
    return params


def extract_docstring_section(docstring, section_name):
    pattern = rf'{section_name}:\s*\n```(?:python|dockerfile|bash|yaml)?\n([\s\S]*?)```'
    match = re.search(pattern, docstring, re.IGNORECASE)
    return match.group(1).strip() if match else ''


def extract_full_section(docstring, section_name):
    pattern = rf'^{section_name}:\s*\n([\s\S]*?)(?=\n[A-Z][A-Z ]+:\s*\n|\Z)'
    match = re.search(pattern, docstring, re.MULTILINE | re.IGNORECASE)
    return match.group(1).strip() if match else ''


def get_cwe_name(cwe_id):
    names = {
        'CWE-78': 'Improper Neutralization of Special Elements used in an OS Command',
        'CWE-79': 'Improper Neutralization of Input During Web Page Generation',
        'CWE-89': "Improper Neutralization of Special Elements used in an SQL Command ('SQL Injection')",
        'CWE-95': 'Improper Neutralization of Directives in Dynamically Evaluated Code',
        'CWE-96': 'Improper Neutralization of Directives in Statically Saved Code',
        'CWE-200': 'Exposure of Sensitive Information to an Unauthorized Actor',
        'CWE-22': 'Improper Limitation of a Pathname to a Restricted Directory',
        'CWE-250': 'Execution with Unnecessary Privileges',
        'CWE-259': 'Use of Hard-coded Password',
        'CWE-264': 'Permissions, Privileges, and Access Controls',
        'CWE-269': 'Improper Privilege Management',
        'CWE-284': 'Improper Access Control',
        'CWE-287': 'Improper Authentication',
        'CWE-295': 'Improper Certificate Validation',
        'CWE-319': 'Cleartext Transmission of Sensitive Information',
        'CWE-322': 'Key Exchange without Entity Authentication',
        'CWE-326': 'Inadequate Encryption Strength',
        'CWE-327': 'Use of a Broken or Risky Cryptographic Algorithm',
        'CWE-330': 'Use of Insufficiently Random Values',
        'CWE-352': 'Cross-Site Request Forgery (CSRF)',
        'CWE-489': 'Active Debug Code',
        'CWE-502': 'Deserialization of Untrusted Data',
        'CWE-506': 'Embedded Malicious Code',
        'CWE-521': 'Weak Password Requirements',
        'CWE-522': 'Insufficiently Protected Credentials',
        'CWE-532': 'Insertion of Sensitive Information into Log File',
        'CWE-538': 'Insertion of Sensitive Information into Externally-Accessible File or Directory',
        'CWE-601': 'URL Redirection to Untrusted Site',
        'CWE-611': 'Improper Restriction of XML External Entity Reference',
        'CWE-614': 'Sensitive Cookie in HTTPS Session Without Secure Attribute',
        'CWE-668': 'Exposure of Resource to Wrong Sphere',
        'CWE-676': 'Use of Potentially Dangerous Function',
        'CWE-710': 'Improper Adherence to Coding Standards',
        'CWE-732': 'Incorrect Permission Assignment for Critical Resource',
        'CWE-770': 'Allocation of Resources Without Limits or Throttling',
        'CWE-829': 'Inclusion of Functionality from Untrusted Control Sphere',
        'CWE-863': 'Incorrect Authorization',
        'CWE-918': 'Server-Side Request Forgery (SSRF)',
        'CWE-942': 'Permissive Cross-domain Policy with Untrusted Domains',
        'CWE-1078': 'Inappropriate Source Code Style or Formatting',
        'CWE-1236': 'Improper Neutralization of Formula Elements in a CSV File',
        'CWE-1333': 'Inefficient Regular Expression Complexity',
    }
    return names.get(cwe_id, cwe_id)


def extract_preamble(content):
    """Extract imports and class definitions before the first decorator."""
    lines = content.split('\n')
    preamble_lines = []
    in_docstring = False
    past_docstring = False

    for line in lines:
        trimmed = line.strip()

        if not past_docstring:
            if trimmed.startswith('"""') or trimmed.startswith("'''"):
                quote = trimmed[:3]
                if in_docstring:
                    in_docstring = False
                    past_docstring = True
                    continue
                elif trimmed.count(quote) >= 2 and len(trimmed) > 3:
                    past_docstring = True
                    continue
                else:
                    in_docstring = True
                    continue
            if in_docstring:
                continue
            if trimmed and not trimmed.startswith('#'):
                past_docstring = True
            else:
                continue

        if re.match(r'@(dockerfile_rule|compose_rule|python_rule|java_rule)', trimmed):
            break

        preamble_lines.append(line)

    while preamble_lines and preamble_lines[-1].strip() == '':
        preamble_lines.pop()

    return '\n'.join(preamble_lines)


def extract_rule_code(content, match):
    """Extract decorator + function body for a single rule."""
    start = match.start()
    remaining = content[start + len(match.group(0)):]
    next_decorator = re.search(r'\n@(dockerfile_rule|compose_rule|python_rule|java_rule)', remaining)
    end = start + len(match.group(0)) + next_decorator.start() if next_decorator else len(content)
    return content[start:end].strip()


def filter_preamble_for_rule(preamble, rule_code):
    """Keep only imports and class definitions that the rule actually uses."""
    lines = preamble.split('\n')
    kept = []
    i = 0
    while i < len(lines):
        line = lines[i]
        class_match = re.match(r'^class\s+(\w+)', line)
        if class_match:
            class_name = class_match.group(1)
            block = [line]
            i += 1
            while i < len(lines) and (lines[i].startswith('    ') or lines[i].strip() == ''):
                block.append(lines[i])
                i += 1
            if class_name in rule_code:
                kept.extend(block)
        else:
            kept.append(line)
            i += 1

    return re.sub(r'\n{3,}', '\n\n', '\n'.join(kept)).strip()


def migrate_file(filepath, dry_run=False):
    """Migrate all rules in a file to YAML folder structure."""
    content = open(filepath).read()
    matches = list(DECORATOR_PATTERN.finditer(content))

    if not matches:
        return 0

    # Extract module docstring
    docstring_match = re.match(r'^"""([\s\S]*?)"""', content)
    docstring = docstring_match.group(1).strip() if docstring_match else ''

    # Extract shared preamble (imports + classes)
    preamble = extract_preamble(content)

    # Extract examples from docstring (shared across rules in the file)
    vulnerable_code = extract_docstring_section(docstring, 'VULNERABLE EXAMPLE')
    secure_code = extract_docstring_section(docstring, 'SECURE EXAMPLE')
    if not secure_code:
        secure_code = extract_docstring_section(docstring, 'SECURE ALTERNATIVES')

    description = extract_full_section(docstring, 'DESCRIPTION')
    refs_text = extract_full_section(docstring, 'REFERENCES')
    references = []
    for line in refs_text.split('\n'):
        line = line.strip().lstrip('- ')
        if line:
            references.append({'title': line, 'url': ''})

    dirpath = os.path.dirname(filepath)
    count = 0

    for match in matches:
        decorator_type = match.group(1)
        decorator_content = match.group(2)
        func_name = match.group(3)
        params = parse_decorator_params(decorator_content)

        rule_id = params.get('id', '')
        if not rule_id:
            continue

        # Determine language
        lang_map = {
            'dockerfile_rule': 'docker',
            'compose_rule': 'docker-compose',
            'python_rule': 'python',
            'java_rule': 'java',
        }
        language = lang_map.get(decorator_type, 'unknown')

        # Extract this rule's code
        rule_code_body = extract_rule_code(content, match)

        # Filter preamble to only include classes this rule uses
        filtered_preamble = filter_preamble_for_rule(preamble, rule_code_body)
        full_rule_code = f'{filtered_preamble}\n\n\n{rule_code_body}' if filtered_preamble else rule_code_body

        # Build CWE
        cwe_id = params.get('cwe', '')
        cwe_list = []
        if cwe_id:
            cwe_list.append({
                'id': cwe_id,
                'name': get_cwe_name(cwe_id),
                'url': f'https://cwe.mitre.org/data/definitions/{cwe_id.replace("CWE-", "")}.html'
            })

        # Build tags
        tags = [t.strip() for t in params.get('tags', '').split(',') if t.strip()]

        category = params.get('category', os.path.basename(dirpath))

        # Build meta
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
            'references': references,
            'compliance': [],
            'faq': [],
            'similar_rules': [],
            'tests': {'positive': 'tests/positive/', 'negative': 'tests/negative/'},
        }

        if params.get('owasp'):
            meta['owasp'] = [{'id': params['owasp'], 'name': '', 'url': ''}]

        out_dir = os.path.join(dirpath, rule_id)

        if dry_run:
            print(f'  DRY  {rule_id}')
            count += 1
            continue

        os.makedirs(os.path.join(out_dir, 'tests', 'positive'), exist_ok=True)
        os.makedirs(os.path.join(out_dir, 'tests', 'negative'), exist_ok=True)

        with open(os.path.join(out_dir, 'rule.py'), 'w') as f:
            f.write(full_rule_code + '\n')

        with open(os.path.join(out_dir, 'meta.yaml'), 'w') as f:
            yaml.dump(meta, f, default_flow_style=False, allow_unicode=True,
                      sort_keys=False, width=100)

        # Write test files
        if vulnerable_code:
            ext = 'Dockerfile' if language == 'docker' else ('docker-compose.yml' if language == 'docker-compose' else 'vulnerable.py')
            with open(os.path.join(out_dir, 'tests', 'positive', ext), 'w') as f:
                f.write(vulnerable_code + '\n')

        if secure_code:
            ext = 'Dockerfile' if language == 'docker' else ('docker-compose.yml' if language == 'docker-compose' else 'safe.py')
            with open(os.path.join(out_dir, 'tests', 'negative', ext), 'w') as f:
                f.write(secure_code + '\n')

        print(f'  OK   {rule_id}')
        count += 1

    return count


def main():
    dry_run = '--dry-run' in sys.argv
    paths = [a for a in sys.argv[1:] if a != '--dry-run']

    if not paths:
        print('Usage: migrate-rules-to-yaml.py [--dry-run] <file_or_dir> ...')
        sys.exit(1)

    total = 0
    for path in paths:
        if os.path.isfile(path):
            total += migrate_file(path, dry_run)
        elif os.path.isdir(path):
            for root, dirs, files in os.walk(path):
                if 'meta.yaml' in files:
                    dirs.clear()
                    continue
                for f in sorted(files):
                    if f.endswith('.py') and not f.startswith('__'):
                        total += migrate_file(os.path.join(root, f), dry_run)

    print(f'\n{"Would migrate" if dry_run else "Migrated"}: {total} rules')


if __name__ == '__main__':
    main()
