package network

// Mode represents the network access mode for a sandbox.
type Mode string

const (
	ModeNone       Mode = "none"
	ModeRestricted Mode = "restricted"
	ModeOpen       Mode = "open"
)

// IsAllowed checks if a host/domain is allowed in the given mode.
func (m Mode) IsAllowed(host string) bool {
	switch m {
	case ModeNone:
		return false
	case ModeOpen:
		return true
	case ModeRestricted:
		return DefaultAllowlist.Contains(host)
	default:
		return false
	}
}

// IsDefaultDeny checks if a host is in the default deny list.
func IsDefaultDeny(host string) bool {
	return DefaultDeny.Contains(host)
}

// DefaultDeny is the list of hosts that are always blocked.
var DefaultDeny = DomainList{
	"169.254.169.254", // Cloud metadata
	"169.254.20.1",    // AWS instance metadata
	"100.169.90.56",   // AWS internal
	"127.0.0.1",       // Host localhost
	"localhost",
	"0.0.0.0",
}

// DefaultAllowlist is the default domain allowlist for restricted mode.
var DefaultAllowlist = DomainList{
	"github.com",
	"api.github.com",
	"raw.githubusercontent.com",
	"registry.npmjs.org",
	"registry.yarnpkg.com",
	"pypi.org",
	"files.pythonhosted.org",
	"proxy.golang.org",
	"golang.org",
	"storage.googleapis.com",
	"crates.io",
	"static.crates.io",
	"packages.microsoft.com",
	"download.docker.com",
}

// DomainList is a set of domain names.
type DomainList []string

// Contains checks if a domain is in the list.
func (d DomainList) Contains(domain string) bool {
	for _, allowed := range d {
		if allowed == domain {
			return true
		}
		// Also match subdomains: domain ends with ".allowed"
		if len(domain) > len(allowed)+1 {
			if domain[len(domain)-len(allowed)-1:] == "."+allowed {
				return true
			}
		}
	}
	return false
}
