package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Users maps a case-sensitive user name to its PHC hash.
type Users map[string]string

// LoadUsers reads lines of `name:<phc>`; `#` comments and blank lines are skipped.
// Anything but an argon2id hash is rejected so a plaintext password never goes live.
func LoadUsers(path string) (Users, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("users file %q cannot be read (%v): create it with one `name:hash` line per user, see deploy/README.md", path, err)
	}
	defer f.Close()
	users := Users{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, hash, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("users file %q line %d: expected `name:hash`", path, n)
		}
		if name == "" {
			return nil, fmt.Errorf("users file %q line %d: empty user name", path, n)
		}
		if !strings.HasPrefix(hash, "$argon2id$") {
			return nil, fmt.Errorf("users file %q line %d: the value for %q is not an argon2id hash (plaintext passwords are not allowed): create one with `hash-password`", path, n, name)
		}
		if _, dup := users[name]; dup {
			return nil, fmt.Errorf("users file %q line %d: duplicate user name %q", path, n, name)
		}
		users[name] = hash
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("users file %q: %v", path, err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("users file %q contains no users: add at least one `name:hash` line", path)
	}
	return users, nil
}
