package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	now := time.Now()
	fmt.Println(now)

	// load different time zone
	timeLocation, _ := time.LoadLocation("Asia/Dhaka")
	currentTimeInDhaka := time.Now().In(timeLocation)
	fmt.Println(currentTimeInDhaka)
	fmt.Println("Week day number :", int(currentTimeInDhaka.Weekday()), "=>", currentTimeInDhaka.Weekday())

	timeFormat := "2006-01-02 15:04:05"
	formatedDateTime := currentTimeInDhaka.Format(timeFormat)

	dateTimeArray := strings.Split(formatedDateTime, " ")
	formatedDate := dateTimeArray[0]
	formatedTime := dateTimeArray[1]
	fmt.Println(formatedDate)
	fmt.Println(formatedTime)
	fmt.Println(formatedDateTime)

	db, err := sql.Open("mysql", "root:@/test")
	checkErr(err)
	// query
	var record_time string
	err = db.QueryRow("SELECT record_time FROM gps_data WHERE code = 'GTVT0001'").Scan(&record_time)
	checkErr(err)
	fmt.Println(record_time)
	//close the db connection
	db.Close()
	startTime, _ := time.Parse(timeFormat, record_time)
	endTime, _ := time.Parse(timeFormat, formatedDateTime)
	fmt.Println(timeDifferenceInMinutes(startTime, endTime))
}

func timeDifferenceInMinutes(startTime time.Time, endTime time.Time) float64 {
	duration := endTime.Sub(startTime)
	diffInSecond := duration.Minutes()
	return diffInSecond
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
