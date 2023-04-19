package main

import "fmt"

func usage() {
	fmt.Println("Welcome to Gocker!")
	fmt.Println("Supported commands:")
	fmt.Println("gocker run [--mem] [--swap] [--pids] [--cpus] <image> <command>")
}