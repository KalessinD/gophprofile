package config

import (
	"strings"
)

// ParseConfigPath extracts config file path from args without using flag package
func ParseConfigPath(args []string) string {
	for i, arg := range args {
		switch {
		case arg == "-c", arg == "-config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "-c="):
			return strings.TrimPrefix(arg, "-c=")
		case strings.HasPrefix(arg, "-config="):
			return strings.TrimPrefix(arg, "-config=")
		}
	}
	return ""
}
