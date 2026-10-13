package main

var callback = func() error { return nil }
var counter int

func init() {}

func init() {
	counter++
}

func main() {
	if counter == 0 {
		counter++
	}
}
