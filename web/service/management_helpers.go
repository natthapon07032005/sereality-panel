package service

import "fmt"

func requireRowsAffected(resource string, rows int64) error {
	if rows == 0 {
		return fmt.Errorf("%s not found", resource)
	}
	return nil
}
