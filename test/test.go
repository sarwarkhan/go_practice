package main

import "fmt"
import "curl"

func main() {
	fmt.Println("Hello, World!")

	lat, lng := "-33.8670", "151.1957"
	location, err := curl.GetLocationAddress(lat, lng)
	checkErr(err)
	fmt.Println(location)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
