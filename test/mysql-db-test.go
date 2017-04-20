package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:@/test")
	checkErr(err)

	// insert
	stmt, err := db.Prepare("INSERT gps_data SET code=?, record_date=?, record_time=?, data_status=?, engine_status=?, speed=?")
	checkErr(err)

	res, err := stmt.Exec("GTVT0001", "2017-04-18", "2017-04-18 10:33:45", 1, 1, 80.50)
	checkErr(err)

	id, err := res.LastInsertId()
	checkErr(err)

	fmt.Println(id)
	// update
	stmt, err = db.Prepare("update gps_data set speed=? where id=?")
	checkErr(err)

	res, err = stmt.Exec(71.03, id)
	checkErr(err)

	affect, err := res.RowsAffected()
	checkErr(err)

	fmt.Println(affect)

	// query
	rows, err := db.Query("SELECT * FROM gps_data")
	checkErr(err)

	for rows.Next() {
		var id int
		var code string
		var record_date string
		var record_time string
		var data_status string
		var engine_status string
		var speed float64
		err = rows.Scan(&id, &code, &record_date, &record_time, &data_status, &engine_status, &speed)
		checkErr(err)
		fmt.Println(id)
		fmt.Println(code)
		fmt.Println(record_date)
		fmt.Println(record_time)
		fmt.Println(data_status)
		fmt.Println(engine_status)
		fmt.Println(speed)
	}

	// delete
	//stmt, err = db.Prepare("delete from userinfo where uid=?")
	//checkErr(err)

	//res, err = stmt.Exec(id)
	//checkErr(err)

	//affect, err = res.RowsAffected()
	//checkErr(err)

	//fmt.Println(affect)

	db.Close()

}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
