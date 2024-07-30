package gokubedownscaler

import (
	"flag"
	"fmt"
	"os"
)

func GetEnvValue(key string, value flag.Value) error {
	if val, ok := os.LookupEnv(key); ok {
		err := value.Set(val)
		if err != nil {
			return fmt.Errorf("failed to set value: %w", err)
		}
	}
	return nil
}
