package main

import (
	"fmt"
	"strings"
)

func greet(name string) string {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		name = "World"
	}

	return fmt.Sprintf("Hello, %s!", name)
}
