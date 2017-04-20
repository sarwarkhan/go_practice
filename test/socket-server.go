package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	port := ":8081"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println("Server up and listening on port :" + port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	for {
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return
		}
		incomingData := strings.Split(line, ",")

		terminal := strings.Split(incomingData[0], "#")
		terminalID := terminal[0]
		fmt.Printf("%q\n", terminalID)

		speed := knotToKm(incomingData[7])
		fmt.Println(speed)

		sensor := strings.Split(incomingData[12], "|")
		sensorData := sensor[1]
		fmt.Printf("%s\n", sensorData)

		engineStatus := sensorData[:1]
		fmt.Printf("%s\n", engineStatus)

		acStatus := sensorData[1:2]
		fmt.Printf("%s\n", acStatus)
	}
}

func knotToKm(knotValue string) float64 {
	kmValue, err := strconv.ParseFloat(knotValue, 64)
	if err == nil {
		kmValue = kmValue * 1.852
	}

	return kmValue
}
