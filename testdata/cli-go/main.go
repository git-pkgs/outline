package main

import (
	"os/exec"

	"example.com/svc/store"
)

func Handler(name string) error {
	rec, err := store.Load(name)
	if err != nil {
		return err
	}
	return exec.Command(rec.Path).Run()
}

func main() {
	_ = Handler("x")
}
