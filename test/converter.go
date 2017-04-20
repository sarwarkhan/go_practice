package main

import (
	"fmt"
	"strconv"
)

func main() {
	speed, err := knotToKm("45.25")
	checkErr(err)
	fmt.Println(speed)

	latitude, err := decimalMinute("2347.104884")
	checkErr(err)
	fmt.Println(latitude)
	longitude, err := decimalMinute("09022.102628")
	checkErr(err)
	fmt.Println(longitude)
}

func knotToKm(val string) (float64, error) {
	decimal, err := strconv.ParseFloat(val, 64)
	decimal = decimal * 1.852

	return decimal, err
}

func decimalMinute(val string) (float64, error) {
	f, err := strconv.ParseFloat(val, 64)
	f = f / 100.00

	g := int(f)
	gi := float64(g)
	mm := f - gi
	result := gi + (mm*100.00)/60.00

	return result, err
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
