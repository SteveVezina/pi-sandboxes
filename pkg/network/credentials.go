package network

import (
	"encoding/base64"

	"github.com/pi-sandbox/pi/pkg/secrets"
)

// CredentialInjectorFromStore builds a CredentialInjector that, for an
// approved host, resolves every matching credential rule to its in-memory
// value and turns it into an Authorization header (T30.8).
//
//   - git-token:     Basic base64("x-access-token:<token>")  (GitHub/GitLab HTTPS git)
//   - registry-auth: Basic base64("<name|token>:<secret>")
//
// The secret value never leaves this function except on the injected
// header to the approved host.
func CredentialInjectorFromStore(s *secrets.CredentialStore) CredentialInjector {
	return func(host string) []HeaderInjection {
		if s == nil {
			return nil
		}
		var out []HeaderInjection
		for _, c := range s.GetForHost(host) {
			v, err := s.Resolve(c.ID)
			if err != nil || v == "" {
				continue
			}
			switch c.Type {
			case "git-token":
				out = append(out, HeaderInjection{"Authorization", "Basic " + basicAuth("x-access-token", v)})
			case "registry-auth":
				user := c.Name
				if user == "" {
					user = "token"
				}
				out = append(out, HeaderInjection{"Authorization", "Basic " + basicAuth(user, v)})
			}
		}
		return out
	}
}

func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}
