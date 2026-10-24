package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnv reads a .env file and populates process environment variables.
func LoadEnv(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err // file optional in production if using system env vars
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
	return scanner.Err()
}
