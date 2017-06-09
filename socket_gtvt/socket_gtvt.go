package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

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
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	for {
		SMS_INTERVAL := 10.00 //in minutes
		SMS_HOST_USER := ""
		SMS_HOST_PASS := ""
		SMS_SENDER := ""
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return
		}
		// load Bangladesh time zone
		timeLocation, _ := time.LoadLocation("Asia/Dhaka")
		currentTimeInDhaka := time.Now().In(timeLocation)
		//special consideration
		timeFormat := "2006-01-02 15:04:05"
		formatedDateTime := currentTimeInDhaka.Format(timeFormat)
		dateTimeArray := strings.Split(formatedDateTime, " ")
		formatedDate := dateTimeArray[0]
		formatedTime := dateTimeArray[1]
		weekDay := int(currentTimeInDhaka.Weekday())
		/* start test */
		fmt.Printf("formatedDate: %s, Type: %T\n", formatedDate, formatedDate)
		fmt.Printf("formatedTime: %s, Type: %T\n", formatedTime, formatedTime)
		fmt.Printf("formatedDateTime: %s, Type: %T\n", formatedDateTime, formatedDateTime)
		fmt.Printf("weekDay: %d, Type: %T\n", weekDay, weekDay)
		/* end test */
		//close the socket connection with the client
		conn.Close()
		/*strat processing client data*/
		incomingData := strings.Split(line, ",")
		if len(incomingData) != 13 {
			return
		}
		//get terminalID or device_emei
		terminal := strings.Split(incomingData[0], "#")
		if len(terminal) != 2 {
			return
		}
		terminalID := terminal[0]
		fmt.Printf("terminalID: %s, Type: %T\n", terminalID, terminalID)
		//get data type or status
		dataType := incomingData[2] // 'A' = Active data, 'V' = void data
		fmt.Printf("dataType: %s, Type: %T\n", dataType, dataType)
		//get latitude
		latitudeString := incomingData[3]
		latitudeInDecimalMinutes, latitudeConvertError := decimalMinute(latitudeString)
		if latitudeConvertError != nil {
			return
		}
		fmt.Printf("latitudeInDecimalMinutes: %f, Type: %T\n", latitudeInDecimalMinutes, latitudeInDecimalMinutes)
		//get longitude
		longitudeString := incomingData[5]
		longitudeInDecimalMinutes, longitudeConvertError := decimalMinute(longitudeString)
		if longitudeConvertError != nil {
			return
		}
		fmt.Printf("longitudeInDecimalMinutes: %f, Type: %T\n", longitudeInDecimalMinutes, longitudeInDecimalMinutes)
		//get east-west-north-south indicator
		n_s_indicator := incomingData[4]
		e_w_indicator := incomingData[6]
		//decide the vehicle direction
		movingDirection := ""
		if n_s_indicator == "N" {
			movingDirection = "north"
		} else {
			movingDirection = "south"
		}
		if e_w_indicator == "E" {
			movingDirection += "-east"
		} else {
			movingDirection += "-west"
		}
		fmt.Printf("movingDirection: %s, Type: %T\n", movingDirection, movingDirection)
		//get speed
		speedInKnot := incomingData[7]
		speedInDecimal, speedConvertError := knotToKm(speedInKnot)
		if speedConvertError != nil {
			return
		}
		fmt.Printf("speedInDecimal: %f, Type: %T\n", speedInDecimal, speedInDecimal)
		//get bearing
		bearing := incomingData[8]
		fmt.Printf("bearing: %s, Type: %T\n", bearing, bearing)
		//get sensor data
		sensor := strings.Split(incomingData[12], "|")
		if len(sensor) != 3 {
			return
		}
		sensorData := sensor[1]
		//get engine status
		engineStatus := sensorData[:1]
		fmt.Printf("engineStatus: %s, Type: %T\n", engineStatus, engineStatus)
		//get ac status
		acStatus := sensorData[1:2]
		fmt.Printf("acStatus: %s, Type: %T\n", acStatus, acStatus)
		//db connection
		db, dbError := sql.Open("mysql", "root:@/test")
		if dbError != nil {
			return
		}
		//insert into gps_data_gtvt table
		insertSQL := "INSERT gps_data_gtvt SET device_emei=?, record_date=?, record_time=?,"
		insertSQL += " latitude=?, longitude=?, n_s_indicator=?, e_w_indicator=?, speed=?,"
		insertSQL += " bearing=?, direction=?, engine_status=?, data_status=?, ac_status=?"
		//prepared statement
		preparedStmt, stmtError := db.Prepare(insertSQL)
		if stmtError != nil {
			db.Close()
			return
		}
		//execute prepared statement
		dbResult, execError := preparedStmt.Exec(terminalID, formatedDate, formatedDateTime,
			latitudeInDecimalMinutes, longitudeInDecimalMinutes, n_s_indicator, e_w_indicator, speedInDecimal,
			bearing, movingDirection, engineStatus, dataType, acStatus)
		if execError != nil {
			db.Close()
			return
		}
		/* process next steps */
		vehicleSelectSQL := "SELECT D.device_id, D.emei_number, VDM.vehicle_id,"
		vehicleSelectSQL += " V.call_back_sim, V.number_plate, V.vehicle_owner_id, V.speed_limit,"
		vehicleSelectSQL += " V.vehicle_is_active, V.is_overspeed_sms, V.is_geofence_sms, V.is_destination_sms,"
		vehicleSelectSQL += " U.user_id, U.user_is_active, U.remaining_sms,"
		vehicleSelectSQL += " SM.sms_year, SM.sms_month, SM.sms_total, SM.sms_used FROM devices AS D"
		vehicleSelectSQL += " LEFT JOIN vehicle_device_mapping AS VDM ON VDM.device_id = D.device_id"
		vehicleSelectSQL += " LEFT JOIN vehicles AS V ON V.vehicle_id = VDM.vehicle_id"
		vehicleSelectSQL += " LEFT JOIN users AS U ON U.user_id = V.vehicle_owner_id"
		vehicleSelectSQL += " LEFT JOIN sms_monthly AS SM ON SM.user_id = U.user_id"
		vehicleSelectSQL += " WHERE D.emei_number=? AND SM.sms_year=? AND SM.sms_month=? LIMIT 1"
		var _deviceId, _vehicleId, _vehicleOwnerId, _isVehicleActive, _isOverspeedSMS, _isGeofenceSMS, _isDestinationSMS, _userId, _isUserActive, _smsRemain, _smsYear, _smsMonth, _smsTotal, _smsUsed int
		var _emeiNumber, _callBackSim, _numberPlate string
		var _speedLimit float64
		//select vehicle record
		vSelectError := db.QueryRow(vehicleSelectSQL, terminalID, currentTimeInDhaka.Year(), int(currentTimeInDhaka.Month())).Scan(&_deviceId, &_emeiNumber, &_vehicleId, &_callBackSim, &_numberPlate, &_vehicleOwnerId, &_speedLimit, &_isVehicleActive, &_isOverspeedSMS, &_isGeofenceSMS, &_isDestinationSMS, &_userId, &_isUserActive, &_smsRemain, &_smsYear, &_smsMonth, &_smsTotal, &_smsUsed)
		if vSelectError != nil {
			db.Close()
			return
		}
		geofenceSMSStatus := "NA" //need to relocate
		//stop processing incase of inactive vehicle and user
		if _isVehicleActive == 0 || _isUserActive == 0 {
			db.Close()
			return
		}
		//stop processing if no sms remains
		if _smsUsed >= _smsTotal && _smsRemain <= 0 {
			db.Close()
			return
		}
		//stop processing incase of invalid call-back sim number
		if _callBackSim == nil || len(_callBackSim) == 0 {
			db.Close()
			return
		}

		// process over speed alarm
		if _speedLimit < speedInDecimal {
			if _isOverspeedSMS == 1 {
				var speedSMSSendingTime string
				smsLogSelectError := db.QueryRow("SELECT sending_time FROM sms_log WHERE sms_type = 'OVER_SPEED' AND vehicle_id=? ORDER BY sending_time DESC LIMIT 1", _vehicleId).Scan(&speedSMSSendingTime)
				if smsLogSelectError == nil {
					elegibleForOverSpeedSMS := false
					if speedSMSSendingTime == nil {
						elegibleForOverSpeedSMS = true
					} else {
						startTime, _ := time.Parse(timeFormat, speedSMSSendingTime)
						endTime, _ := time.Parse(timeFormat, formatedDateTime)
						previousSMSSendingTime := timeDifferenceInMinutes(startTime, endTime)
						if previousSMSSendingTime >= SMS_INTERVAL {
							elegibleForOverSpeedSMS = true
						}
					}
					if elegibleForOverSpeedSMS {
						isSMSLogUpdated := updateSMSLog()
					}
				}
			}
		}

	}
}

func updateSMSLog(uId int, vId int, smsYear int, smsMonth int, smsTotal int, smsUsed int, smsRemain int, smsType string, sms string, callBackSim string, exeDatetime string, geofenceSMSStatus string) bool {
	smsLogFlag := false
	return smsLogFlag
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

func knotToKm(knotValue string) (float64, error) {
	kmValue, err := strconv.ParseFloat(knotValue, 64)
	kmValue = kmValue * 1.852

	return kmValue, err
}

func timeDifferenceInMinutes(startTime time.Time, endTime time.Time) float64 {
	duration := endTime.Sub(startTime)
	diffInSecond := duration.Minutes()
	return diffInSecond
}
