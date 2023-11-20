package util

import "fmt"

func Must(err error, message string) {
	if err != nil {
		panic(fmt.Errorf(message+" : %w", err))
	}
}
