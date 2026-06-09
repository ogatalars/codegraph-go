package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Mode string // "cli" | "mcp"
	Root string
}

// Wizard prompts the user for mode and project root.
func Wizard() (*Config, error) {
	r := bufio.NewReader(os.Stdin)

	fmt.Println("codegraph-go — code intelligence for AI\n")

	mode, err := prompt(r, "Mode [cli/mcp] (default: cli): ", "cli")
	if err != nil {
		return nil, err
	}
	mode = strings.ToLower(mode)
	if mode != "cli" && mode != "mcp" {
		mode = "cli"
	}

	root, err := prompt(r, "Project root (default: ./): ", "./")
	if err != nil {
		return nil, err
	}

	return &Config{Mode: mode, Root: root}, nil
}

func prompt(r *bufio.Reader, question, defaultVal string) (string, error) {
	fmt.Print(question)
	line, err := r.ReadString('\n')
	if err != nil {
		return defaultVal, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}
