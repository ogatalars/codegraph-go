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

func Wizard() (*Config, *bufio.Reader, error) {
	r := bufio.NewReader(os.Stdin)

	fmt.Println("codegraph-go — code intelligence for AI")
	fmt.Println()

	mode, err := prompt(r, "Mode [cli/mcp] (default: cli): ", "cli")
	if err != nil {
		return nil, nil, err
	}
	mode = strings.ToLower(mode)
	if mode != "cli" && mode != "mcp" {
		mode = "cli"
	}

	root, err := prompt(r, "Project root (default: ./): ", "./")
	if err != nil {
		return nil, nil, err
	}

	return &Config{Mode: mode, Root: root}, r, nil
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
