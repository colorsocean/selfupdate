package main

import (
	"time"

	. "github.com/colorsocean/selfupdate/uptool"
)

func main() {
	defer Recover()

	//IssuerExeIsAService(true)
	StopIssuerExeServiceWithin(60 * time.Second)
	RemoveIssuerExeWithin(1 * time.Second)
	ReplaceIssuerExeWith("test.exe")
	Done()
}
