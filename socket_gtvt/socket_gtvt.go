package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"pip"
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
		SMS_API := fmt.Sprint("http://app.planetgroupbd.com/api/sendsms/plain?user=", SMS_HOST_USER, "&password=", SMS_HOST_PASS, "&sender=", SMS_SENDER)
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
			movingDirection = fmt.Sprint(movingDirection, "-east")
		} else {
			movingDirection = fmt.Sprint(movingDirection, "-west")
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
		insertSQL = fmt.Sprint(insertSQL, " latitude=?, longitude=?, n_s_indicator=?, e_w_indicator=?, speed=?,")
		insertSQL = fmt.Sprint(insertSQL, " bearing=?, direction=?, engine_status=?, data_status=?, ac_status=?")
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
		} else {
			_ = dbResult
		}
		//stop further processing for void data
		if dataType == "V" {
			db.Close()
			return
		}

		/* process next steps */
		vehicleSelectSQL := "SELECT D.device_id, D.emei_number, VDM.vehicle_id,"
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " V.call_back_sim, V.number_plate, V.vehicle_owner_id, V.speed_limit,")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " V.vehicle_is_active, V.is_overspeed_sms, V.is_geofence_sms, V.is_destination_sms,")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " U.user_id, U.user_is_active, U.remaining_sms,")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " SM.sms_year, SM.sms_month, SM.sms_total, SM.sms_used FROM devices AS D")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " LEFT JOIN vehicle_device_mapping AS VDM ON VDM.device_id = D.device_id")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " LEFT JOIN vehicles AS V ON V.vehicle_id = VDM.vehicle_id")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " LEFT JOIN users AS U ON U.user_id = V.vehicle_owner_id")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " LEFT JOIN sms_monthly AS SM ON SM.user_id = U.user_id")
		vehicleSelectSQL = fmt.Sprint(vehicleSelectSQL, " WHERE D.emei_number=? AND SM.sms_year=? AND SM.sms_month=? LIMIT 1")
		var _deviceId, _vehicleId, _vehicleOwnerId, _isVehicleActive, _isOverspeedSMS, _isGeofenceSMS, _isDestinationSMS, _userId, _isUserActive, _smsRemain, _smsYear, _smsMonth, _smsTotal, _smsUsed int
		var _emeiNumber, _callBackSim, _numberPlate string
		var _speedLimit float64
		//select vehicle record
		vSelectError := db.QueryRow(vehicleSelectSQL, terminalID, currentTimeInDhaka.Year(), int(currentTimeInDhaka.Month())).Scan(&_deviceId, &_emeiNumber, &_vehicleId, &_callBackSim, &_numberPlate, &_vehicleOwnerId, &_speedLimit, &_isVehicleActive, &_isOverspeedSMS, &_isGeofenceSMS, &_isDestinationSMS, &_userId, &_isUserActive, &_smsRemain, &_smsYear, &_smsMonth, &_smsTotal, &_smsUsed)
		if vSelectError != nil {
			db.Close()
			return
		}
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
		if len(_callBackSim) == 0 {
			db.Close()
			return
		}

		//imp variable assignment
		fixedSpeed := toFixed(speedInDecimal)
		geofenceSMSStatus := "NA"
		smsApiUrl := ""
		isSMSLogUpdated := false
		locationAddress := ""
		dataProcessingTime, _ := time.Parse(timeFormat, formatedDateTime)

		// process over speed alarm
		if _speedLimit < speedInDecimal {
			if _isOverspeedSMS == 1 {
				textMessageSpeed := fmt.Sprint("SPEED Limit violation. V: ", _numberPlate, ", SPEED: ", fixedSpeed, " km/h at ")
				var speedSMSSendingTime string
				smsLogSelectError := db.QueryRow("SELECT sending_time FROM sms_log WHERE sms_type = 'OVER_SPEED' AND vehicle_id=? ORDER BY sending_time DESC LIMIT 1", _vehicleId).Scan(&speedSMSSendingTime)
				if smsLogSelectError == nil {
					elegibleForOverSpeedSMS := false
					if len(speedSMSSendingTime) == 0 {
						elegibleForOverSpeedSMS = true
					} else {
						speedSMSStartTime, _ := time.Parse(timeFormat, speedSMSSendingTime)
						previousSpeedSMSSendingTime := timeDifferenceInMinutes(speedSMSStartTime, dataProcessingTime)
						if previousSpeedSMSSendingTime >= SMS_INTERVAL {
							elegibleForOverSpeedSMS = true
						}
					}
					if elegibleForOverSpeedSMS {
						locationForSpeedAlart, addressError := getLocationAddress(latitudeInDecimalMinutes, longitudeInDecimalMinutes)
						locationAddress = locationForSpeedAlart
						if addressError == nil {
							textMessageSpeed = fmt.Sprint(textMessageSpeed, locationAddress)
							isSMSLogUpdated = updateSMSLog(db, _userId, _vehicleId, _smsYear, _smsMonth, _smsTotal, _smsUsed, _smsRemain, "OVER_SPEED", textMessageSpeed, _callBackSim, formatedDateTime, geofenceSMSStatus)
							_smsUsed++
							_smsRemain--
							if isSMSLogUpdated {
								if len(textMessageSpeed) > 160 {
									textMessageSpeed = textMessageSpeed[:159]
								}
								speedSmsText := url.QueryEscape(textMessageSpeed)
								smsApiUrl = fmt.Sprint(SMS_API, "&SMSText=", speedSmsText, "&GSM=", _callBackSim)
								//send sms
								http.Get(smsApiUrl)
							}
						}
					}
				}
			}
		}

		// process geo-fence alarm
		if _isGeofenceSMS == 1 {
			textMessageGeofenceOUT := fmt.Sprint("GEO-FENCE violation. V: ", _numberPlate, ", SPEED: ", fixedSpeed, " km/h at ")
			textMessageGeofenceIN := fmt.Sprint("Inside GEO-FENCE. V: ", _numberPlate, ", SPEED: ", fixedSpeed, " km/h at ")
			var geofenceSMSSendingTime string
			geofenceSMSLogSelectError := db.QueryRow("SELECT sending_time, geofence_sms_status FROM sms_log WHERE sms_type = 'GEO_FENCE' AND vehicle_id=? ORDER BY sending_time DESC LIMIT 1", _vehicleId).Scan(&geofenceSMSSendingTime, &geofenceSMSStatus)
			if geofenceSMSLogSelectError == nil {
				geofenceINFlag := false
				geofenceOUTFlag := false
				if len(geofenceSMSSendingTime) == 0 {
					geofenceOUTFlag = true
				} else {
					geofenceSMSStartTime, _ := time.Parse(timeFormat, geofenceSMSSendingTime)
					previousGeofenceSMSSendingTime := timeDifferenceInMinutes(geofenceSMSStartTime, dataProcessingTime)
					if previousGeofenceSMSSendingTime >= SMS_INTERVAL {
						if geofenceSMSStatus == "IN" {
							geofenceOUTFlag = true
						} else if geofenceSMSStatus == "OUT" {
							geofenceINFlag = true
						}
					}
				}

				if geofenceINFlag || geofenceOUTFlag {
					//process geo-fence violation
					weekDayFlag := false
					geofenceCoordinates := ""

					geofenceScheduleSelectSQL := "SELECT GFS.geofence_id, GFS.vehicle_id, GFS.week_day, GFS.start_time, GFS.end_time, GFS.is_active, GFS.geofence_coordinates"
					geofenceScheduleSelectSQL = fmt.Sprint(geofenceScheduleSelectSQL, " FROM geo_fence_schedules AS GFS")
					geofenceScheduleSelectSQL = fmt.Sprint(geofenceScheduleSelectSQL, " LEFT JOIN geo_fence AS GF ON GF.geofence_id = GFS.geofence_id")
					geofenceScheduleSelectSQL = fmt.Sprint(geofenceScheduleSelectSQL, " WHERE GFS.is_active = 1 AND GFS.vehicle_id = ? AND (? BETWEEN GFS.start_time AND GFS.end_time)")
					geofenceSchedules, scheduleError := db.Query(geofenceScheduleSelectSQL, _vehicleId, dataProcessingTime)
					if scheduleError == nil {
						var _geofenceId, _geofenceIsActive int
						var _weekDay, _geofenceScheduleStartTime, _geofenceScheduleEndTime string
						for geofenceSchedules.Next() {
							geofenceScheduleError := geofenceSchedules.Scan(&_geofenceId, &_vehicleId, &_weekDay, &_geofenceScheduleStartTime, &_geofenceScheduleEndTime, &_geofenceIsActive, &geofenceCoordinates)
							if geofenceScheduleError == nil {
								scheduledDay, _ := strconv.Atoi(_weekDay)
								if len(geofenceCoordinates) > 0 && (scheduledDay == 7 || scheduledDay == weekDay) {
									weekDayFlag = true
									break
								}
							}
						}
					}
					if weekDayFlag {
						vertics := []pip.Point{}
						fence := strings.Split(geofenceCoordinates, "|")
						// complete the fence by pushing first vertics
						fence = append(fence, fence[0])

						for i := 0; i < len(fence); i++ {
							pointString := strings.Split(fence[i], ",")
							xAxis, _ := strconv.ParseFloat(pointString[0], 64)
							yAxis, _ := strconv.ParseFloat(pointString[1], 64)
							vertics = append(vertics, pip.Point{X: xAxis, Y: yAxis})
						}
						geofencePloygon := pip.Polygon{
							vertics,
						}
						checkPoint := pip.Point{X: latitudeInDecimalMinutes, Y: longitudeInDecimalMinutes}
						insideGeofence := pip.PointInPolygon(checkPoint, geofencePloygon) //false=outside | true=inside geofence

						geofenceAlartMessage := ""
						geofenceAlartSMSFlag := false
						if len(locationAddress) == 0 {
							locationAddressForGeofenceAlart, addressError := getLocationAddress(latitudeInDecimalMinutes, longitudeInDecimalMinutes)
							if addressError != nil {
								db.Close()
								return
							}
							locationAddress = locationAddressForGeofenceAlart
						}
						if geofenceINFlag && insideGeofence {
							textMessageGeofenceIN = fmt.Sprint(textMessageGeofenceIN, locationAddress)
							if len(textMessageGeofenceIN) > 160 {
								geofenceAlartMessage = textMessageGeofenceIN[:159]
							} else {
								geofenceAlartMessage = textMessageGeofenceIN
							}
							//update the flag
							geofenceAlartSMSFlag = true
							geofenceSMSStatus = "IN"
						} else if geofenceOUTFlag && insideGeofence == false {
							textMessageGeofenceOUT = fmt.Sprint(textMessageGeofenceOUT, locationAddress)
							if len(textMessageGeofenceOUT) > 160 {
								geofenceAlartMessage = textMessageGeofenceOUT[:159]
							} else {
								geofenceAlartMessage = textMessageGeofenceOUT
							}
							//update the flag
							geofenceAlartSMSFlag = true
							geofenceSMSStatus = "OUT"
						}
						if geofenceAlartSMSFlag {
							isSMSLogUpdated = updateSMSLog(db, _userId, _vehicleId, _smsYear, _smsMonth, _smsTotal, _smsUsed, _smsRemain, "OVER_SPEED", geofenceAlartMessage, _callBackSim, formatedDateTime, geofenceSMSStatus)
							if isSMSLogUpdated {
								geofenceSmsText := url.QueryEscape(geofenceAlartMessage)
								smsApiUrl = fmt.Sprint(SMS_API, "&SMSText=", geofenceSmsText, "&GSM=", _callBackSim)
								//send sms
								http.Get(smsApiUrl)
							}
						}
					}
				}
			}
		}
		//make sure db connection is closed
		db.Close()
	}
}

func updateSMSLog(_db *sql.DB, uId int, vId int, smsYear int, smsMonth int, smsTotal int, smsUsed int, smsRemain int, smsType string, sms string, callBackSim string, exeDatetime string, smsStatus string) bool {
	smsLogFlag := false
	smsUsedFrom := "MONTHLY"
	updateSQL := ""
	if smsUsed < smsTotal { //deduct sms from user monthly sms
		updateSQL = "UPDATE sms_monthly SET sms_used = (sms_used + 1) WHERE user_id = ? AND sms_year = ? and sms_month = ?"
		updateStmt, _err := _db.Prepare(updateSQL)
		result, _err := updateStmt.Exec(uId, smsYear, smsMonth)
		if _err != nil {
			return false
		} else {
			_ = result
		}
		smsLogFlag = true
	} else if smsUsed >= smsTotal && smsRemain > 0 {
		smsUsedFrom = "RESERVED"
		updateSQL = "UPDATE users SET remaining_sms = (remaining_sms - 1) WHERE user_id = ?"
		updateStmt, _err := _db.Prepare(updateSQL)
		result, _err := updateStmt.Exec(uId)
		if _err != nil {
			return false
		} else {
			_ = result
		}
		smsLogFlag = true
	}

	//insert into sms_log table
	if smsLogFlag {
		smsLogInsertSQL := "INSERT INTO sms_log (vehicle_id, sms_type, receipent, sms, sending_time, sending_type, sms_used_from, geofence_sms_status) VALUES (?,?,?,?,?,?,?,?)"
		insertStmt, _err := _db.Prepare(smsLogInsertSQL)
		result, _err := insertStmt.Exec(vId, smsType, callBackSim, sms, exeDatetime, "AUTO", smsUsedFrom, smsStatus)
		if _err != nil {
			return false
		} else {
			_ = result
		}
	}

	return smsLogFlag
}

func getLocationAddress(lat float64, lng float64) (string, error) {
	formatedAddress := ""
	url := fmt.Sprint("https://maps.googleapis.com/maps/api/geocode/json?latlng=", lat, ",", lng)
	getResp, err := http.Get(url)
	if err != nil {
		return formatedAddress, err
	}
	defer getResp.Body.Close()

	resp := new(Response)
	if getResp.StatusCode == 200 { // OK
		err = json.NewDecoder(getResp.Body).Decode(resp)
	}
	if err != nil {
		return formatedAddress, err
	} else {
		formatedAddress = resp.GoogleResponse.Results[0].Address
	}
	return formatedAddress, nil
}

type Response struct {
	*GoogleResponse
}

type GoogleResponse struct {
	Results []*GoogleResult
}

type GoogleResult struct {
	Address string `json:"formatted_address"`
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

func toFixed(num float64) float64 {
	return float64(int(num*100+0.5)) / 100
}

func timeDifferenceInMinutes(startTime time.Time, endTime time.Time) float64 {
	duration := endTime.Sub(startTime)
	diffInSecond := duration.Minutes()
	return diffInSecond
}
