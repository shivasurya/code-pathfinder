# PyPI Publishing Setup

This workflow supports two authentication methods for publishing to PyPI. **Trusted Publishing is recommended** as it's more secure and doesn't require managing secrets.

## Option 1: PyPI Trusted Publishing (Recommended) ✅

**No GitHub secrets required!**

Trusted Publishing uses OpenID Connect (OIDC) for secure, token-less authentication.

### Setup Steps:

1. **Go to PyPI Project Settings**
   - Visit: https://pypi.org/manage/project/codepathfinder/settings/publishing/
   - Or navigate: PyPI → Your Projects → codepathfinder → Settings → Publishing

2. **Add GitHub as a Trusted Publisher**
   - Click "Add a new publisher"
   - Fill in the form:
     - **Owner**: `shivasurya`
     - **Repository name**: `code-pathfinder`
     - **Workflow name**: `pypi-publish.yml`
     - **Environment name**: `pypi`
   - Click "Add"

3. **Create GitHub Environment** (if not exists)
   - Go to: https://github.com/shivasurya/code-pathfinder/settings/environments
   - Click "New environment"
   - Name it: `pypi`
   - (Optional) Add protection rules:
     - Required reviewers
     - Wait timer
     - Deployment branches (only tags: `v*`)

4. **Done!**
   - No secrets to manage
   - Automatic token rotation
   - More secure than API tokens

### How It Works

The workflow uses `id-token: write` permission to request a short-lived token from GitHub, which PyPI validates and accepts for publishing.

---

## Option 2: API Token (Alternative)

If you prefer using traditional API tokens instead of Trusted Publishing:

### Setup Steps:

1. **Generate PyPI API Token**
   - Visit: https://pypi.org/manage/account/token/
   - Click "Add API token"
   - Name: "GitHub Actions - code-pathfinder"
   - Scope: "Project: codepathfinder" (after first manual upload)
   - Copy the token (starts with `pypi-`)

2. **Add GitHub Secret**
   - Go to: https://github.com/shivasurya/code-pathfinder/settings/secrets/actions
   - Click "New repository secret"
   - Name: `PYPI_API_TOKEN`
   - Value: (paste the token)
   - Click "Add secret"

3. **Update Workflow**
   - Replace the publish step in `.github/workflows/pypi-publish.yml`:

   ```yaml
   - name: Publish to PyPI
     uses: pypa/gh-action-pypi-publish@release/v1
     with:
       packages-dir: dist/
       password: ${{ secrets.PYPI_API_TOKEN }}  # Add this line
   ```

4. **Remove Trusted Publishing Config**
   - Remove `id-token: write` permission
   - Remove `environment: pypi` from publish job

---

## Current Configuration

The workflow is currently configured for **Trusted Publishing (Option 1)**.

### Files Involved

- **Workflow**: `.github/workflows/pypi-publish.yml`
- **Publish job**: Uses `id-token: write` for OIDC
- **Environment**: `pypi` (configure in GitHub repo settings)

---

## Testing Before Publishing

### Dry Run (Manual Trigger)

Test the workflow without publishing:

1. Go to Actions → "Publish to PyPI" → "Run workflow"
2. Select branch: `main`
3. **Version**: `1.1.3` (or your version)
4. **Skip tests**: Leave unchecked
5. **Publish to PyPI**: **Uncheck this** ✅ (dry run mode)
6. Click "Run workflow"

This will:
- ✅ Build all platform wheels
- ✅ Run installation tests
- ✅ Create all artifacts
- ❌ Skip PyPI upload

### Full Run (Actual Publishing)

When ready to publish:

**Via Tag Push** (automatic):
```bash
git tag v1.1.3
git push origin v1.1.3
```

**Via Manual Trigger**:
1. Go to Actions → "Publish to PyPI" → "Run workflow"
2. **Version**: `1.1.3`
3. **Publish to PyPI**: Keep checked ✅
4. Click "Run workflow"

---

## Troubleshooting

### Error: "Trusted publisher configuration does not match"

- Verify the workflow name is exactly `pypi-publish.yml`
- Check the environment name is exactly `pypi`
- Ensure repository owner matches

### Error: "OIDC token validation failed"

- Make sure `id-token: write` permission is set in the workflow
- Verify GitHub Actions has permissions enabled in repo settings

### Error: "Invalid or expired token"

If using API Token (Option 2):
- Regenerate the PyPI token
- Update the `PYPI_API_TOKEN` secret in GitHub

---

## Security Best Practices

✅ **DO:**
- Use Trusted Publishing (Option 1)
- Require manual approval for production deployments
- Use environment protection rules
- Limit deployment to tags only (`v*`)

❌ **DON'T:**
- Store API tokens in code
- Use account-wide PyPI tokens (use project-scoped)
- Skip environment protection for production

---

## References

- [PyPI Trusted Publishers Guide](https://docs.pypi.org/trusted-publishers/)
- [GitHub Actions OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [PyPA Publish Action](https://github.com/pypa/gh-action-pypi-publish)
