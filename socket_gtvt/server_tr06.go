package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	_ "time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	port := ":9781"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println("Server up and listening on port :" + port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("couldn't accept: " + err.Error())
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	//conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	//SMS_INTERVAL := 10.00 //in minutes
	//SMS_HOST_USER := ""
	//SMS_HOST_PASS := ""
	//SMS_SENDER := ""
	//SMS_API := fmt.Sprint("http://app.planetgroupbd.com/api/sendsms/plain?user=", SMS_HOST_USER, "&password=", SMS_HOST_PASS, "&sender=", SMS_SENDER)
	for {
		inData, err := bufio.NewReader(conn).ReadBytes('\r')
		if err != nil {
			fmt.Println("Error in reading ...")
			return
		}
		line := hex.EncodeToString(inData)
		fmt.Println(line)

	}
}
