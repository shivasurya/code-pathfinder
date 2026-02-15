"""
GO-SEC-004: Hardcoded Credentials in Source Code

VULNERABILITY DESCRIPTION:
Hardcoded credentials (passwords, API keys, tokens, secrets) in source code pose
a critical security risk. These credentials can be extracted from repositories,
binaries, or version control history.

SEVERITY: HIGH
CWE: CWE-798 (Use of Hard-coded Credentials)
OWASP: A07:2021 (Identification and Authentication Failures)

IMPACT:
- Credential theft from repository access
- API key exposure in version control history
- Unauthorized access to services
- Difficulty in credential rotation
- Compliance violations (PCI DSS, SOC 2, HIPAA)

VULNERABLE PATTERNS:
Variable names containing:
- password, passwd, pwd
- secret, secretkey
- api_key, apikey, api-key
- token, auth_token, access_token
- credential, credentials

SECURE PATTERNS:
- Use environment variables: os.Getenv("API_KEY")
- Use secret management systems (AWS Secrets Manager, HashiCorp Vault)
- Use configuration files excluded from version control
- Use encrypted configuration with runtime decryption

EXAMPLE:
Vulnerable:
    apiKey := "sk-1234567890abcdef"  // Hardcoded
    password := "super_secret_123"

Secure:
    apiKey := os.Getenv("API_KEY")
    password := getSecretFromVault("db_password")

REFERENCES:
- https://owasp.org/www-community/vulnerabilities/Use_of_hard-coded_password
- https://cwe.mitre.org/data/definitions/798.html
"""

from codepathfinder import rule, variable

@rule(
    id="GO-SEC-004",
    severity="HIGH",
    cwe="CWE-798",
    owasp="A07:2021"
)
def go_hardcoded_credentials():
    """Detects hardcoded passwords, API keys, and tokens in Go source code."""
    return variable(
        pattern="*password*|*secret*|*api_key*|*apikey*|*token*|*credential*"
    )
