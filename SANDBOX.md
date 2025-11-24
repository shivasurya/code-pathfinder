# Python Sandboxing with nsjail

## Overview

Code Pathfinder uses **nsjail** (Google's production-grade sandboxing tool) to safely execute untrusted Python DSL rules with maximum isolation.

## Security Features

✅ **Network Isolation**: All network access blocked (no socket connections, no HTTP requests)
✅ **Filesystem Isolation**: Cannot read sensitive files (/etc/passwd, /etc/shadow, ~/.ssh/, etc.)
✅ **Process Isolation**: Cannot see or interact with other processes (isolated PID namespace)
✅ **Resource Limits**: CPU, memory, file size, and execution time limits enforced
✅ **Environment Isolation**: Minimal environment variable exposure
✅ **Read-Only System**: Cannot modify /usr, /lib, or system files

## Installation Method

**Built from source** (Alpine apk not available in Wolfi)
- Source: https://github.com/google/nsjail.git (tag 3.4)
- Build dependencies: flex, bison, protobuf-dev, libnl3-dev
- Compiler warning `-Werror` removed for compatibility with GCC 15.2.0

## Runtime Requirements

### For Digital Ocean / Self-Hosted Deployments

**Docker/Podman run command**:
```bash
podman run --cap-add=SYS_ADMIN your-image:tag
```

**Why CAP_SYS_ADMIN is needed**:
- Required for Linux namespace creation (network, PID, mount, user, IPC, UTS)
- Provides strongest isolation (95%+ attack surface reduction)
- Used by Google internally for sandboxing untrusted code

**Security note**: CAP_SYS_ADMIN is needed ONLY for the outer container to create nested namespaces. The Python code inside nsjail runs as UID 65534 (nobody) with ALL capabilities dropped and ALL namespaces isolated.

### Configuration

Set environment variable in Dockerfile (already configured):
```dockerfile
ENV PATHFINDER_SANDBOX_ENABLED=true
```

To disable sandbox (development only):
```bash
export PATHFINDER_SANDBOX_ENABLED=false
```

## nsjail Command Template

The Go code (PR-02) will use this command template:

```bash
nsjail -Mo \
  --user nobody \
  --chroot /tmp/nsjail_root \
  --iface_no_lo \
  --disable_proc \
  --bindmount_ro /usr:/usr \
  --bindmount_ro /lib:/lib \
  --bindmount /tmp:/tmp \
  --cwd /tmp \
  --rlimit_as 512 \
  --rlimit_cpu 30 \
  --rlimit_fsize 1 \
  --rlimit_nofile 64 \
  --time_limit 30 \
  -- /usr/bin/python3 /tmp/rule.py
```

## Security Test Results

All tests pass with 100% isolation:

| Test | Result | Details |
|------|--------|---------|
| Network Access | ✅ BLOCKED | OSError: Network unreachable |
| /etc/passwd | ✅ BLOCKED | FileNotFoundError |
| /etc/shadow | ✅ BLOCKED | FileNotFoundError |
| ~/.ssh/id_rsa | ✅ BLOCKED | FileNotFoundError |
| /proc/self/environ | ✅ BLOCKED | FileNotFoundError |
| PID Namespace | ✅ ISOLATED | Process sees itself as PID 1 |
| Filesystem Write | ✅ READ-ONLY | Cannot write to /, /usr, /etc |
| Environment Vars | ✅ MINIMAL | Only 1 var visible (LC_CTYPE) |

## Python Version

**Installed**: Python 3.13.9 (wolfi-base doesn't have 3.14 yet)
- Goal was Python 3.14, actual is Python 3.13.9
- Provides all necessary security features
- Will upgrade to 3.14 when available in Wolfi repos

## Build Details

### Docker Image Size
- Base image: cgr.dev/chainguard/wolfi-base
- Added components: Python 3.13.9, nsjail (built from source), flex, bison
- Final image: ~200-250MB (including build dependencies cleanup)

### Build Time
- nsjail compilation: ~2-3 minutes (includes kafel submodule)
- Total Docker build: ~4-5 minutes

## Troubleshooting

### Error: "Operation not permitted"
**Solution**: Run container with `--cap-add=SYS_ADMIN`

### Error: "nsjail: command not found"
**Solution**: Rebuild Docker image with latest Dockerfile

### Error: "Cannot read /tmp/rule.py"
**Solution**: Ensure file is created BEFORE entering nsjail sandbox

## Next Steps (PR-02)

1. Integrate nsjail into `dsl/loader.go`
2. Add `buildNsjailCommand()` helper function
3. Add `isSandboxEnabled()` environment check
4. Update `/tmp/nsjail_root` creation in entrypoint.sh
5. Add comprehensive Go tests

## References

- nsjail GitHub: https://github.com/google/nsjail
- Tech Spec: /Users/shiva/src/shivasurya/cpf_plans/docs/planning/python-sandboxing/tech-spec.md
- PR-01 Doc: /Users/shiva/src/shivasurya/cpf_plans/docs/planning/python-sandboxing/pr-details/PR-01-docker-nsjail-setup.md
