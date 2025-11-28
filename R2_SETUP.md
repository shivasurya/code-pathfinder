# Cloudflare R2 Setup for Stdlib Registries

This document contains the configuration needed to set up Cloudflare R2 for hosting Python stdlib registries at `assets.codepathfinder.dev`.

## ✅ What's Been Done

- Created upload script: `sourcecode-parser/tools/upload_to_r2.sh`
- Created GitHub Action workflow: `.github/workflows/stdlib-r2-upload.yml`
- Updated version to `1.0.0`
- Updated all URLs to use `assets.codepathfinder.dev`
- Tested generation locally (8.2MB for Python 3.14, 190 modules)
- Removed outdated Cloudflare Pages deploy workflow

## 📋 What You Need to Configure

### 1. Cloudflare R2 Bucket Setup

**Bucket Name:** `code-pathfinder-assets`

**Configuration:**
- Region: Auto (Cloudflare handles this)
- Public Access: Enabled via custom domain
- CORS: Allow all origins (for browser access)

### 2. Custom Domain Configuration

**Domain:** `assets.codepathfinder.dev`

**DNS Setup:**
1. Go to Cloudflare DNS settings
2. Add CNAME record:
   - Name: `assets`
   - Target: `<your-r2-bucket-endpoint>` (provided by Cloudflare)
   - Proxy: Enabled (orange cloud)

**R2 Custom Domain:**
1. Go to R2 bucket settings
2. Add custom domain: `assets.codepathfinder.dev`
3. Enable public access

### 3. API Credentials

You need to create R2 API tokens for the GitHub Action workflow.

**Steps:**
1. Go to Cloudflare Dashboard → R2 → Manage R2 API Tokens
2. Create API Token with:
   - Name: `code-pathfinder-github-actions`
   - Permissions: `Edit` (for upload/sync)
   - Scope: Bucket `code-pathfinder-assets`

You'll receive:
- **Access Key ID** - like AWS access key
- **Secret Access Key** - like AWS secret key
- **Account ID** - your Cloudflare account ID

### 4. GitHub Secrets Configuration

Add these secrets to your GitHub repository:

**Navigate to:** `Settings` → `Secrets and variables` → `Actions` → `New repository secret`

Add the following secrets:

| Secret Name | Value | Description |
|-------------|-------|-------------|
| `R2_ACCOUNT_ID` | Your Cloudflare Account ID | Found in R2 dashboard |
| `R2_ACCESS_KEY_ID` | Your R2 Access Key ID | From API token creation |
| `R2_SECRET_ACCESS_KEY` | Your R2 Secret Access Key | From API token creation |

## 🧪 Testing Locally (Optional)

If you want to test the upload script locally before using GitHub Actions:

```bash
# Set environment variables
export R2_ACCOUNT_ID="your-account-id"
export R2_ACCESS_KEY_ID="your-access-key-id"
export R2_SECRET_ACCESS_KEY="your-secret-access-key"

# Install AWS CLI if not already installed
# brew install awscli  # macOS
# apt-get install awscli  # Linux

# Run the upload script
cd sourcecode-parser/tools
./upload_to_r2.sh
```

## 📦 What Gets Uploaded

**Directory Structure in R2:**
```
code-pathfinder-assets/
└── registries/
    ├── python3.11/
    │   └── stdlib/
    │       └── v1/
    │           ├── manifest.json
    │           ├── os_stdlib.json
    │           ├── sys_stdlib.json
    │           └── ... (all stdlib modules)
    ├── python3.12/
    │   └── stdlib/v1/... (similar structure)
    └── python3.14/
        └── stdlib/v1/... (similar structure)
```

**Size Estimates:**
- Python 3.11: ~8-10 MB (190 modules)
- Python 3.12: ~8-10 MB (190 modules)
- Python 3.14: ~8.2 MB (190 modules)
- **Total: ~25-30 MB**

Well within Cloudflare R2's 10 GB free tier! ✅

## 🚀 How It Works

### GitHub Action Trigger

The workflow runs automatically when:
1. A new release is published (after binaries are built)
2. Manual workflow dispatch (for testing)

### Workflow Steps

1. Checkout code
2. Setup Python 3.11, 3.12, and 3.14
3. Configure AWS CLI for R2
4. Run `upload_to_r2.sh` which:
   - Generates stdlib registries for each Python version
   - Validates JSON files
   - Uploads to R2 using `aws s3 sync --delete`
5. Verify uploads
6. Test public accessibility

### URL Structure

After upload, registries will be available at:
- `https://assets.codepathfinder.dev/registries/python3.11/stdlib/v1/manifest.json`
- `https://assets.codepathfinder.dev/registries/python3.12/stdlib/v1/manifest.json`
- `https://assets.codepathfinder.dev/registries/python3.14/stdlib/v1/manifest.json`

## ✅ Verification Checklist

After setting up R2 and adding GitHub secrets:

- [ ] R2 bucket `code-pathfinder-assets` created
- [ ] Custom domain `assets.codepathfinder.dev` configured
- [ ] Public access enabled on R2 bucket
- [ ] R2 API token created with Edit permissions
- [ ] GitHub secrets added (`R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`)
- [ ] Test manual workflow dispatch
- [ ] Verify public URLs are accessible

## 🔄 Ongoing Maintenance

### When to Regenerate

Regenerate stdlib registries when:
- New Python version is released (add to `PYTHON_VERSIONS` in scripts)
- Generator improvements (better type inference)
- Bug fixes in introspection

### How to Regenerate

**Option 1: GitHub Action (Recommended)**
- Go to Actions tab → "Upload Stdlib Registries to R2"
- Click "Run workflow" → Select branch → Run

**Option 2: Local Upload**
```bash
export R2_ACCOUNT_ID="..."
export R2_ACCESS_KEY_ID="..."
export R2_SECRET_ACCESS_KEY="..."
cd sourcecode-parser/tools
./upload_to_r2.sh
```

## 💰 Cost Estimate

**Storage:** 30 MB / 10 GB free tier = **$0**
**Operations:** ~100/month / 1M free = **$0**
**Egress:** Unlimited free = **$0**

**Total Monthly Cost: $0** ✅

## 📚 Related Files

- Upload script: `sourcecode-parser/tools/upload_to_r2.sh`
- Generator: `sourcecode-parser/tools/generate_stdlib_registry.py`
- Test script: `sourcecode-parser/tools/test_generation_local.sh`
- GitHub Action: `.github/workflows/stdlib-r2-upload.yml`
- Go client: `sourcecode-parser/graph/callgraph/registry/stdlib_remote.go`
- Builder integration: `sourcecode-parser/graph/callgraph/builder/builder.go`

## 🐛 Troubleshooting

### Upload fails with "Access Denied"
- Verify R2 API token has Edit permissions
- Check GitHub secrets are correctly set
- Ensure token scope includes the correct bucket

### Public URLs return 404
- Verify custom domain is configured in R2
- Check DNS CNAME record is set
- Wait a few minutes for DNS propagation

### Generation fails for Python version
- Ensure Python version is installed on runner
- Check `PYTHON_VERSIONS` array in scripts
- Windows-only modules (msvcrt, winreg) will fail on Linux/macOS (this is expected)

## 📞 Support

For issues with:
- **Cloudflare R2 setup:** Check Cloudflare documentation or support
- **GitHub Actions:** Review workflow logs in Actions tab
- **Code/scripts:** Open an issue on GitHub
