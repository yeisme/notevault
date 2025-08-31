package main

import "github.com/yeisme/notevault/pkg/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
