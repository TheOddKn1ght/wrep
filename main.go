package main

import (
	"fmt"
	"flag"
)

func main() {
	fmt.Println("Hello")
	city :=	flag.String("city", "Moscow", "specify the city for the weather forecast")
	flag.Parse()
	fmt.Println(*city)
}
