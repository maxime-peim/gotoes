package main

import "github.com/maxime-peim/gotoes/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
