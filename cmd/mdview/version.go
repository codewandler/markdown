package main

import "fmt"

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func versionString() string {
	c := commit
	if c == "" {
		c = "unknown"
	}
	d := date
	if d == "" {
		d = "unknown"
	}
	return fmt.Sprintf("mdview %s (commit %s, built %s)", version, c, d)
}
