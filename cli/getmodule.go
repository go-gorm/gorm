package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)


func getModuleName(baseFolder string) (string, error) {
	file, err := os.Open(baseFolder + "/go.mod")
	if err != nil {
		return "", fmt.Errorf("cannot open go.mod: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %v", err)
	}

	return "", fmt.Errorf("module name not found in go.mod")
}