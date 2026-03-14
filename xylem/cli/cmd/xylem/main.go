package main

import (
	"errors"
	"fmt"
	"os"
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit code %d", e.code)
}

func (e *exitError) Unwrap() error {
	return e.err
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			if ee.err != nil {
				fmt.Fprintln(os.Stderr, ee.err)
			}
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
