package main

import (
	"fmt"
	"pip"
	"strconv"
	"strings"
)

func main() {
	vertics := []pip.Point{}
	fence := strings.Split("1.0,1.0|1.0,2.0|2.0,2.0|2.0,1.0", "|")
	// complete the fence by pushing first vertics
	fence = append(fence, fence[0])
	for i := 0; i < len(fence); i++ {
		pointString := strings.Split(fence[i], ",")
		xAxis, err := strconv.ParseFloat(pointString[0], 64)
		yAxis, err := strconv.ParseFloat(pointString[1], 64)
		checkErr(err)

		vertics = append(vertics, pip.Point{X: xAxis, Y: yAxis})
	}
	fmt.Println(vertics)
	ploy := pip.Polygon{
		vertics,
	}

	pt1 := pip.Point{X: 1.1, Y: 4.1}
	insideGeo := pip.PointInPolygon(pt1, ploy)
	fmt.Println(insideGeo)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
