package main

import "github.com/sfi2k7/tinyq"

func main() {
	q := tinyq.NewTinyQ(tinyq.Options{Appname: "smsdos", Port: 9876})
	defer q.Close()

	q.Serve()
}
