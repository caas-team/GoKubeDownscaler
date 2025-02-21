package util

import (
	"fmt"
	"strconv"
)

type Int32Value int32

func (i *Int32Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 32)
	if err != nil {
		return fmt.Errorf("failed to parse as int32: %w", err)
	}
	// #nosec G115
	*i = Int32Value(v)

	return nil
}

func (i *Int32Value) String() string { return strconv.Itoa(int(*i)) }
