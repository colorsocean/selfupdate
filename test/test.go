package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/colorsocean/selfupdate"
)

var (
	exit = make(chan interface{}, 1)
)

func main() {
	defer selfupdate.Init(selfupdate.Options{
		Version:           "0.2",
		MaxUpdateAttempts: 3,
	})()

	go func() {
		defer close(exit)

		<-time.After(2 * time.Second)

		fmt.Println("Getting update")
		resp, err := http.Get("http://localhost:5678/update")
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		w, err := selfupdate.ViaWriteCloser("0.3", true)
		if err != nil {
			panic(err)
		}
		defer w.Close()

		fmt.Println("Writing update")
		io.Copy(w, resp.Body)
	}()

	go func() {
		http.HandleFunc("/update", func(w http.ResponseWriter, req *http.Request) {
			updateFile, err := os.Open("update-2.zip")
			if err != nil {
				panic(err)
			}
			defer updateFile.Close()

			io.Copy(w, updateFile)
		})

		err := http.ListenAndServe(":5678", nil)
		if err != nil {
			fmt.Println("ListenAndServe error: ", err)
			os.Exit(1)
		}
	}()

	<-exit
}
