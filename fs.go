package tinyq

import (
	"fmt"
	"os"
)

func createifnotexists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return fmt.Errorf("Failed to create path %s: %w", path, err)
		}
	}
	return nil
}
