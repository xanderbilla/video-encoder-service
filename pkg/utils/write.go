package utils

import (
	"fmt"
	"os"
)

// WriteFile writes data to a file
func WriteFile(filePath string, data []byte) error {
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
