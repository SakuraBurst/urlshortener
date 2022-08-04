package main

import (
	"fmt"
	"net/url"
)

func main() {
	a, _ := url.Parse("https://www.google.com/")
	fmt.Printf("%+v\n", *a)
	//api.InitAPI()
}
