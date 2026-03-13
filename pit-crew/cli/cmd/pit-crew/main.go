package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"github.com/gofrs/flock"
)

func main() {
	fmt.Println("pit-crew CLI")
	_ = yaml.Node{}
	_ = flock.New("")
}
