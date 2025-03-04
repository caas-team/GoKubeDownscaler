package util

import (
	"flag"
	"fmt"
	"os"
)

// GetEnvValue gets the env value and puts it in flag.Value.
func GetEnvValue(key string, value flag.Value) error {
	if val, ok := os.LookupEnv(key); ok {
		err := value.Set(val)
		if err != nil {
			return NewEnvValueError("GetEnvValue", fmt.Sprintf("failed to set value: %w", err))
		}
	}

	return nil
}
