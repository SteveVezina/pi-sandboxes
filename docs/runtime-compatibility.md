# Runtime Compatibility Guide

## Overview

Pi Sandbox supports three runtime backends, ordered by security level:

| Backend | Security | Speed | Requirements |
|---------|----------|-------|--------------|
| **gVisor** (secure) | 9/10 | Medium | Linux kernel, `runsc` |
| **Fast** (namespaces) | 5/10 | Fast | Linux kernel |
| **Compat** (OCI) | 3/10 | Slow | Docker, Podman, runc, or containerd |

## Auto-Detection

The runtime selection follows a priority order:

1. **gVisor** — Full syscall filtering, user-space kernel
2. **Fast** — Linux namespaces (cgroup, mount, PID, network)
3. **Compat** — OCI container runtime (Docker/Podman/runc)

The first available backend is used automatically:

```go
import "github.com/pi-sandbox/pi/pkg/runtime/detect"

rt, err := detect.Detect("/var/run/pi-sandbox")
if err != nil {
    log.Fatalf("No runtime available: %v", err)
}
fmt.Printf("Using %s backend (mode: %s, security: %d/10)\n",
    rt.Name(), rt.GetMode(), rt.GetSecurityLevel())
```

## Backend Details

### gVisor (Secure)

- **Package**: `pkg/runtime/gvisor`
- **Security Level**: 9/10
- **Mode**: `secure`
- **Requirements**:
  - Linux kernel 4.14+
  - `runsc` installed (`apt install runsc`)
  - Kernel config: `CONFIG_SECCOMP`, `CONFIG_USER_NS`
- **Features**:
  - Full syscall filtering
  - User-space kernel emulation
  - Namespace isolation
  - Resource limits via cgroup v2

### Fast Backend

- **Package**: `pkg/runtime/fast`
- **Security Level**: 5/10
- **Mode**: `fast`
- **Requirements**:
  - Linux kernel 4.14+
  - No additional packages needed
- **Features**:
  - User namespace isolation
  - Mount namespace
  - PID namespace
  - cgroup v2 resource limits
  - Seccomp-bpf syscall filtering
- **Limitations**:
  - Shared kernel (no syscall filtering)
  - Not suitable for untrusted workloads

### Compat Backend (OCI)

- **Package**: `pkg/runtime/compat`
- **Security Level**: 3/10
- **Mode**: `compat`
- **Requirements**:
  - Docker, Podman, runc, or containerd
- **Features**:
  - Full container isolation
  - OCI-compliant
  - Cross-platform support (Linux, macOS via Docker Desktop)
- **Limitations**:
  - Highest overhead
  - Requires daemon running
  - Shared kernel

## Benchmark Comparison

### Expected Performance

| Operation | gVisor | Fast | Compat |
|-----------|--------|------|--------|
| Warm exec p50 | ~50ms | ~10ms | ~100ms |
| Artifact export | ~200ms | ~100ms | ~500ms |
| Create sandbox | ~200ms | ~50ms | ~1s |
| Destroy sandbox | ~100ms | ~20ms | ~500ms |

### Running Benchmarks

```bash
# Run all benchmarks
pi bench all

# Run specific benchmark
pi bench warm-exec

# Compare backends
pi bench compare --backends=gvisor,fast,compat
```

## Platform Support

| Platform | gVisor | Fast | Compat |
|----------|--------|------|--------|
| Linux (native) | ✅ | ✅ | ✅ |
| macOS | ❌ | ❌ | ✅ (Docker Desktop) |
| Windows | ❌ | ❌ | ✅ (WSL2) |

## Migration Guide

### From Fast to gVisor

1. Install `runsc`:
   ```bash
   curl -fsSL https://gvisor.dev/install.sh | sh
   ```

2. Verify installation:
   ```bash
   runsc --version
   ```

3. Restart the daemon — it will auto-detect gVisor.

### From Compat to Fast

1. Ensure Linux kernel supports namespaces:
   ```bash
   cat /proc/sys/kernel/ns/clonedevices
   ```

2. No additional packages needed.

### From gVisor to Compat

1. Install Docker/Podman
2. Restart daemon — it will fall back to OCI runtime

## Troubleshooting

### "No sandbox runtime available"

1. Check available runtimes:
   ```bash
   pi system status
   ```

2. Install a backend:
   - gVisor: `apt install runsc`
   - Fast: No install needed (Linux only)
   - Compat: `apt install docker-ce` or `apt install podman`

3. Check kernel support:
   ```bash
   zgrep CONFIG_USER_NS /proc/config.gz
   zgrep CONFIG_SECCOMP /proc/config.gz
   ```

### Slow performance

1. Check which backend is in use:
   ```bash
   pi system status
   ```

2. If using Compat, consider upgrading to Fast or gVisor.

3. If using Fast, ensure cgroup v2 is mounted:
   ```bash
   mount | grep cgroup
   ```

## Security Considerations

### gVisor (Recommended for untrusted workloads)

- Full syscall filtering prevents kernel exploitation
- User-space kernel eliminates kernel attack surface
- Suitable for multi-tenant environments

### Fast (Suitable for trusted workloads)

- Namespace isolation prevents filesystem/network access
- cgroup limits prevent resource exhaustion
- Shared kernel means kernel exploits are possible

### Compat (Least secure)

- Container isolation depends on OCI runtime
- Docker/Podman run with root privileges by default
- Not suitable for untrusted workloads without additional hardening

## Future Backends

### Milestone 5: MicroVM (Firecracker/Cloud Hypervisor)

- Security Level: 10/10
- Full hardware virtualization
- Tiny guest rootfs
- Sub-millisecond startup
- Highest isolation
