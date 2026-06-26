//go:build linux
// +build linux

package fast

import (
	"encoding/json"
	"fmt"
	"os"
)

// SeccompProfile defines the seccomp-bpf filter.
type SeccompProfile struct {
	DefaultAction string                  `json:"defaultAction"`
	Architectures []string                `json:"architectures"`
	Syscalls      []SeccompSyscallRule    `json:"syscalls"`
}

// SeccompSyscallRule defines a syscall rule.
type SeccompSyscallRule struct {
	Names  []string        `json:"names"`
	Action string          `json:"action"`
	Errno  *int            `json:"errnoRet,omitempty"`
}

// DefaultSeccompProfile returns the default seccomp profile.
func DefaultSeccompProfile() *SeccompProfile {
	return &SeccompProfile{
		DefaultAction: "SCMP_ACT_ERRNO",
		Architectures: []string{"SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_X32"},
		Syscalls: []SeccompSyscallRule{
			{
				Names: []string{
					"read", "write", "open", "close", "stat", "fstat", "lstat",
					"poll", "lseek", "mmap", "mprotect", "munmap", "brk",
					"rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
					"ioctl", "pread64", "pwrite64", "readv", "writev",
					"access", "pipe", "select", "sched_yield", "mremap",
					"msync", "mincore", "madvise", "dup", "dup2", "nanosleep",
					"getpid", "socket", "connect", "accept", "sendto", "recvfrom",
					"bind", "listen", "getsockname", "getpeername", "setsockopt",
					"getsockopt", "clone", "execve", "exit", "wait4", "kill",
					"fcntl", "truncate", "getcwd", "chdir", "rename", "mkdir",
					"rmdir", "creat", "link", "unlink", "symlink", "readlink",
					"chmod", "chown", "lchown", "umask", "gettimeofday",
					"getrlimit", "getrusage", "sysinfo", "times", "ptrace",
					"getuid", "getgid", "geteuid", "getegid", "setuid", "setgid",
					"getgroups", "setgroups", "setresuid", "getresuid",
					"setresgid", "getresgid", "setpgid", "getpgid", "getsid",
					"getsid", "getppid", "getpgrp", "setsid", "sigaltstack",
					"shmdt", "shmget", "shmat", "mlock", "munlock",
					"shmctl", "semop", "semget", "semctl", "msgsnd", "msgrcv",
					"msgget", "msgctl", "fanotify", "clone3",
				},
				Action: "SCMP_ACT_ALLOW",
			},
			{
				Names: []string{
					"mount", "umount2", "pivot_root", "swapon", "swapoff",
					"reboot", "shutdown", "poweroff", "kexec_load", "kexec_setup",
					"init_module", "delete_module", "finit_module",
					"bpf", "lookup_dcookie", "perf_event_open",
					"quotactl", "landlock", "debugfs",
				},
				Action: "SCMP_ACT_ERRNO",
				Errno:  ptrInt(1),
			},
		},
	}
}

func ptrInt(i int) *int { return &i }

// Save writes the seccomp profile to a file.
func (p *SeccompProfile) Save(path string) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal seccomp profile: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads a seccomp profile from a file.
func Load(path string) (*SeccompProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read seccomp profile: %w", err)
	}
	var profile SeccompProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("unmarshal seccomp profile: %w", err)
	}
	return &profile, nil
}
