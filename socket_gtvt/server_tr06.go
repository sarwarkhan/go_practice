package main

import (
	"bufio"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	port := ":6969"
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
	//SMS_INTERVAL := 10.00 //in minutes
	//SMS_HOST_USER := ""
	//SMS_HOST_PASS := ""
	//SMS_SENDER := ""
	//SMS_API := fmt.Sprint("http://app.planetgroupbd.com/api/sendsms/plain?user=", SMS_HOST_USER, "&password=", SMS_HOST_PASS, "&sender=", SMS_SENDER)
	/* client connection state */
	loginState := false
	locationDataFlag := false
	locationDataLastTime := time.Now()
	terminalId := ""
	dataType := "A"
	/* keeping sensor data record -start  */
	engineStatus := 0
	fuelConnectionStatus := "0"
	gpsTrackingStatus := "1"
	alarmStatus := "00"
	alarmType := "000"
	chargeStatus := "0"
	defenceStatus := "0"
	voltageLevelStatus := "04"
	gsmSignalStrength := "04"
	alarmLanguage := "01"
	acStatus := 0
	movingDirection := ""
	/* Keeping location data record -start */
	dateTimeFormated := ""
	dateFormated := ""
	var latitudeInDecimalMinutes float64
	var longitudeInDecimalMinutes float64
	var speedInDecimal float64
	e_w_indicator := "E"
	n_s_indicator := "N"
	bearing := "0"
	//crlf := []byte("\r\n")
	for {
		line, _, err := bufio.NewReader(conn).ReadLine()
		if err != nil {
			fmt.Println("Error in reading ...")
			return
		}
		data := hex.EncodeToString(line)
		incomingDataPacket := fmt.Sprint(data, "0d0a")
		fmt.Println(incomingDataPacket) //print
		if len(incomingDataPacket) < 30 {
			continue
		}

		startBits := incomingDataPacket[:4]
		if startBits == "7878" { // normal data
			// load Bangladesh time zone
			timeLocation, _ := time.LoadLocation("Asia/Dhaka")
			currentTimeInDhaka := time.Now().In(timeLocation)
			//special consideration
			timeFormat := "2006-01-02 15:04:05"
			dateTimeFormated = currentTimeInDhaka.Format(timeFormat)
			dateTimeArray := strings.Split(dateTimeFormated, " ")
			dateFormated = dateTimeArray[0]
			//timeFormated := dateTimeArray[1]
			//day number of week adjustment with db
			weekDay := int(currentTimeInDhaka.Weekday())
			if weekDay == 0 {
				weekDay = 6
			} else {
				weekDay -= 1
			}

			/* incoming data packet length */
			incomingDataPacketLength := len(incomingDataPacket)
			//incomingDataLength := incomingDataPacket[4:6]   //data length string
			incomingDataProtocol := incomingDataPacket[6:8] //incoming data protocol
			/* Serial No from incoming data */
			serialNoPosition := incomingDataPacketLength - 12
			serialNo := incomingDataPacket[serialNoPosition : serialNoPosition+4]
			/* Error code from incoming data */
			errorCodeStartPosition := incomingDataPacketLength - 8
			incomingDataErrorCode := incomingDataPacket[errorCodeStartPosition : errorCodeStartPosition+4]
			/* stop bits */
			stopBitsPosition := incomingDataPacketLength - 4
			stopBits := incomingDataPacket[stopBitsPosition:]
			/* error code checking string */
			errorCodeCheckStrLength := incomingDataPacketLength - 4 - 8
			strForErrorCode := incomingDataPacket[4 : 4+errorCodeCheckStrLength]

			/* check error code */
			table := MakeTable(CRC16_X_25)
			incomingErrorHex, _ := hex.DecodeString(strForErrorCode)
			incomingDataCRC := Checksum(incomingErrorHex, table)        //Error code in uint16
			crcCheck := strconv.FormatUint(uint64(incomingDataCRC), 16) //Error code in string
			if incomingDataErrorCode != crcCheck {                      //consider as void data
				fmt.Println("** VOID Data")
				dataType = "V"
			}

			/* Handle Login data Terminal id */
			if incomingDataProtocol == "01" && dataType == "A" {
				terminalId = incomingDataPacket[8:24] //8+16 = 24
				fmt.Println(fmt.Sprint("Terminal ID: ", terminalId))
			}

			/* handle DB connection */
			db, dbError := sql.Open("mysql", "root:@/test")
			if dbError != nil {
				fmt.Println("** DB Connection Error")
				continue
			}

			/* handle heart-bit data */
			if incomingDataProtocol == "13" && loginState == true && dataType == "A" {
				terminalStatus := incomingDataPacket[8:10]
				voltageLevelStatus := incomingDataPacket[10:12]
				gsmSignalStrength := incomingDataPacket[12:14]
				alarmStatus := incomingDataPacket[14:16]
				alarmLanguage := incomingDataPacket[16:18]
				/* convert terminal information into binary */
				sensorDataBinary, sensorDataConversionError := hex2Bin(terminalStatus)
				if sensorDataConversionError != nil {
					fmt.Println("-> Sensor data conversion error for data-protocol 13: " + terminalId)
					db.Close()
					continue
				}
				// Update global variable for this connection
				fuelConnectionStatus = sensorDataBinary[:1]
				gpsTrackingStatus = sensorDataBinary[1:2]
				alarmType = sensorDataBinary[2:5]
				chargeStatus = sensorDataBinary[5:6]
				engine, engineErr := strconv.Atoi(sensorDataBinary[6:7])
				if engineErr != nil {
					fmt.Println("-> Engine status error for data-protocol 13: " + terminalId)
					db.Close()
					continue
				}
				engineStatus = engine
				if engineStatus == 0 {
					speedInDecimal = 0.00
				}
				defenceStatus = sensorDataBinary[7:8]

				/* prepare insert query for gps_data_tr06 table */
				if locationDataFlag == true {
					lastLocationDataTimeDiff := timeDifferenceInMinutes(locationDataLastTime, time.Now())
					if lastLocationDataTimeDiff >= 3.00 {
						//update previous location data update time
						locationDataLastTime = time.Now()

						//insert into gps_data_tr06 table
						insertSQL := "INSERT gps_data_tr06 SET device_emei=?, record_date=?, record_time=?,"
						insertSQL = fmt.Sprint(insertSQL, " data_status=?, engine_status=?, speed=?,")
						insertSQL = fmt.Sprint(insertSQL, " latitude=?, longitude=?, n_s_indicator=?,")
						insertSQL = fmt.Sprint(insertSQL, " e_w_indicator=?, bearing=?, direction=?,")
						insertSQL = fmt.Sprint(insertSQL, " ac_status=?, fuel_connection_status=?, gps_tracking_status=?,")
						insertSQL = fmt.Sprint(insertSQL, " alarm_status=?, alarm_type=?, charge_status=?,")
						insertSQL = fmt.Sprint(insertSQL, " defence_status=?, voltage_level=?, gsm_signal_strength=?, alarm_language=?")
						//prepared statement
						preparedStmt, stmtError := db.Prepare(insertSQL)
						if stmtError != nil {
							db.Close()
							continue
						}
						//execute prepared statement
						dbResult, execError := preparedStmt.Exec(terminalId, dateFormated, dateTimeFormated,
							dataType, engineStatus, speedInDecimal,
							latitudeInDecimalMinutes, longitudeInDecimalMinutes, n_s_indicator,
							e_w_indicator, bearing, movingDirection,
							acStatus, fuelConnectionStatus, gpsTrackingStatus,
							alarmStatus, alarmType, chargeStatus,
							defenceStatus, voltageLevelStatus, gsmSignalStrength, alarmLanguage)
						if execError != nil {
							db.Close()
							continue
						} else {
							_ = dbResult
						}
						//make sure db connection is closed
						db.Close()
					}
				}
			}
			/* handle location data */
			if incomingDataProtocol == "12" && loginState == true {
				//update previous location data update time
				locationDataLastTime = time.Now()

				hexDatetime := incomingDataPacket[8:20]
				//quantityOfGPS := incomingDataPacket[20:22]
				hexLatitude := incomingDataPacket[22:30]
				hexLongitude := incomingDataPacket[30:38]
				hexSpeed := incomingDataPacket[38:40]
				hexCourseStatus := incomingDataPacket[40:44]
				//hexMCC := incomingDataPacket[44:48]
				//hexMNC := incomingDataPacket[48:50]
				//hexLAC := incomingDataPacket[50:54]
				//hexCellID := incomingDataPacket[54:60]

				/* location data datetime */
				dateTimeFormated = hex2Datetime(hexDatetime)
				if dateTimeFormated == "" {
					db.Close()
					fmt.Println("Datetime conversion error from hex string")
					continue
				}
				dateFormatedArrayLocation := strings.Split(dateTimeFormated, " ")
				dateFormated = dateFormatedArrayLocation[0]
				/* Calculate latitude and longitude */
				latitudeDecimal, latConverErr := hex2Int(hexLatitude)
				if latConverErr != nil {
					db.Close()
					fmt.Println("Latitude conversion error from hex string")
					continue
				}
				latitudeInDecimalMinutes = (float64(latitudeDecimal) / 30000) / 60
				longitudeDecimal, lonConverErr := hex2Int(hexLongitude)
				if lonConverErr != nil {
					db.Close()
					fmt.Println("Longitude conversion error from hex string")
					continue
				}
				longitudeInDecimalMinutes = (float64(longitudeDecimal) / 30000) / 60
				/* Calculate speed in km/h */
				speedFromHex, speedConverErr := hex2Int(hexSpeed)
				if speedConverErr != nil {
					db.Close()
					fmt.Println("Speed conversion error from hex string")
					continue
				}
				speedInDecimal = float64(speedFromHex)
				/* Calculate course and status */
				byteBinary, courceErr := hex2Bin(hexCourseStatus)
				if courceErr != nil {
					db.Close()
					fmt.Println("Course and status conversion error from hex string")
					continue
				}
				if byteBinary[4:5] == "0" {
					e_w_indicator = "E"
				} else {
					e_w_indicator = "W"
				}
				if byteBinary[5:6] == "0" {
					n_s_indicator = "S"
				} else {
					n_s_indicator = "N"
				}

				/* determine moving direction */
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

				bearingValue, bearingConverError := bin2Int(byteBinary[6:16]) //course in decimal degree
				if bearingConverError == nil {
					bearing = fmt.Sprint(bearingValue)
				}

				//insert into gps_data_tr06 table
				insertSQL := "INSERT gps_data_tr06 SET device_emei=?, record_date=?, record_time=?,"
				insertSQL = fmt.Sprint(insertSQL, " data_status=?, engine_status=?, speed=?,")
				insertSQL = fmt.Sprint(insertSQL, " latitude=?, longitude=?, n_s_indicator=?,")
				insertSQL = fmt.Sprint(insertSQL, " e_w_indicator=?, bearing=?, direction=?,")
				insertSQL = fmt.Sprint(insertSQL, " ac_status=?, fuel_connection_status=?, gps_tracking_status=?,")
				insertSQL = fmt.Sprint(insertSQL, " alarm_status=?, alarm_type=?, charge_status=?,")
				insertSQL = fmt.Sprint(insertSQL, " defence_status=?, voltage_level=?, gsm_signal_strength=?, alarm_language=?")
				//prepared statement
				preparedStmt, stmtError := db.Prepare(insertSQL)
				if stmtError != nil {
					db.Close()
					continue
				}
				//execute prepared statement
				dbResult, execError := preparedStmt.Exec(terminalId, dateFormated, dateTimeFormated,
					dataType, engineStatus, speedInDecimal,
					latitudeInDecimalMinutes, longitudeInDecimalMinutes, n_s_indicator,
					e_w_indicator, bearing, movingDirection,
					acStatus, fuelConnectionStatus, gpsTrackingStatus,
					alarmStatus, alarmType, chargeStatus,
					defenceStatus, voltageLevelStatus, gsmSignalStrength, alarmLanguage)
				if execError != nil {
					db.Close()
					continue
				} else {
					_ = dbResult
				}
				//make sure db connection is closed
				db.Close()

			}
			/* handle alarm data */
			if incomingDataProtocol == "16" && loginState == true {
				//update previous location data update time
				locationDataLastTime = time.Now()

				hexDatetime := incomingDataPacket[8:20]
				//quantityOfGPS := incomingDataPacket[20:22]
				hexLatitude := incomingDataPacket[22:30]
				hexLongitude := incomingDataPacket[30:38]
				hexSpeed := incomingDataPacket[38:40]
				hexCourseStatus := incomingDataPacket[40:44]
				//hexLBSLength := incomingDataPacket[44:46]
				//hexMCC := incomingDataPacket[46:50]
				//hexMNC := incomingDataPacket[50:52]
				//hexLAC := incomingDataPacket[52:56]
				//hexCellID := incomingDataPacket[56:62]

				/* alarm data datetime */
				dateTimeFormated = hex2Datetime(hexDatetime)
				if dateTimeFormated == "" {
					db.Close()
					fmt.Println("Datetime conversion error from hex string")
					continue
				}
				dateFormatedArrayLocation := strings.Split(dateTimeFormated, " ")
				dateFormated = dateFormatedArrayLocation[0]
				/* Calculate latitude and longitude */
				latitudeDecimal, latConverErr := hex2Int(hexLatitude)
				if latConverErr != nil {
					db.Close()
					fmt.Println("Latitude conversion error from hex string")
					continue
				}
				latitudeInDecimalMinutes = (float64(latitudeDecimal) / 30000) / 60
				longitudeDecimal, lonConverErr := hex2Int(hexLongitude)
				if lonConverErr != nil {
					db.Close()
					fmt.Println("Longitude conversion error from hex string")
					continue
				}
				longitudeInDecimalMinutes = (float64(longitudeDecimal) / 30000) / 60
				/* Calculate speed in km/h */
				speedFromHex, speedConverErr := hex2Int(hexSpeed)
				if speedConverErr != nil {
					db.Close()
					fmt.Println("Speed conversion error from hex string")
					continue
				}
				speedInDecimal = float64(speedFromHex)
				/* Calculate course and status */
				byteBinary, courceErr := hex2Bin(hexCourseStatus)
				if courceErr != nil {
					db.Close()
					fmt.Println("Course and status conversion error from hex string")
					continue
				}
				if byteBinary[4:5] == "0" {
					e_w_indicator = "E"
				} else {
					e_w_indicator = "W"
				}
				if byteBinary[5:6] == "0" {
					n_s_indicator = "S"
				} else {
					n_s_indicator = "N"
				}

				/* determine moving direction */
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

				bearingValue, bearingConverError := bin2Int(byteBinary[6:16]) //course in decimal degree
				if bearingConverError == nil {
					bearing = fmt.Sprint(bearingValue)
				}

				/* sensore data */
				terminalStatus := incomingDataPacket[62:64]
				voltageLevelStatus = incomingDataPacket[64:66]
				gsmSignalStrength = incomingDataPacket[66:68]
				alarmStatus = incomingDataPacket[68:70]
				alarmLanguage = incomingDataPacket[70:72]

				/* convert terminal information into binary */
				sensorDataBinary, sensorDataConversionError := hex2Bin(terminalStatus)
				if sensorDataConversionError != nil {
					fmt.Println("-> Sensor data conversion error for data-protocol 16: " + terminalId)
					db.Close()
					continue
				}
				/* update global variable */
				fuelConnectionStatus = sensorDataBinary[:1]
				gpsTrackingStatus = sensorDataBinary[1:2]
				alarmType = sensorDataBinary[2:5]
				chargeStatus = sensorDataBinary[5:6]
				engine, engineErr := strconv.Atoi(sensorDataBinary[6:7])
				if engineErr != nil {
					fmt.Println("-> Engine status error for data-protocol 16: " + terminalId)
					db.Close()
					continue
				}
				engineStatus = engine
				defenceStatus = sensorDataBinary[7:8]

				/* set locationDataFlag to true */
				locationDataFlag = true

				//prepare insert query for gps_data_tr06 table
				insertSQL := "INSERT gps_data_tr06 SET device_emei=?, record_date=?, record_time=?,"
				insertSQL = fmt.Sprint(insertSQL, " data_status=?, engine_status=?, speed=?,")
				insertSQL = fmt.Sprint(insertSQL, " latitude=?, longitude=?, n_s_indicator=?,")
				insertSQL = fmt.Sprint(insertSQL, " e_w_indicator=?, bearing=?, direction=?,")
				insertSQL = fmt.Sprint(insertSQL, " ac_status=?, fuel_connection_status=?, gps_tracking_status=?,")
				insertSQL = fmt.Sprint(insertSQL, " alarm_status=?, alarm_type=?, charge_status=?,")
				insertSQL = fmt.Sprint(insertSQL, " defence_status=?, voltage_level=?, gsm_signal_strength=?, alarm_language=?")
				//prepared statement
				preparedStmt, stmtError := db.Prepare(insertSQL)
				if stmtError != nil {
					db.Close()
					continue
				}
				//execute prepared statement
				dbResult, execError := preparedStmt.Exec(terminalId, dateFormated, dateTimeFormated,
					dataType, engineStatus, speedInDecimal,
					latitudeInDecimalMinutes, longitudeInDecimalMinutes, n_s_indicator,
					e_w_indicator, bearing, movingDirection,
					acStatus, fuelConnectionStatus, gpsTrackingStatus,
					alarmStatus, alarmType, chargeStatus,
					defenceStatus, voltageLevelStatus, gsmSignalStrength, alarmLanguage)
				if execError != nil {
					db.Close()
					continue
				} else {
					_ = dbResult
				}
				//make sure db connection is closed
				db.Close()

			}
			/* write back to client incase of login/heart-bit data */
			if dataType == "A" && (incomingDataProtocol == "01" || incomingDataProtocol == "13") {
				/* prepare response data */
				outgoingDataPacket := startBits                                           // initialize with start bits.
				responseDataLength := "05"                                                //hex represent of decimal 5
				outgoingDataPacket = fmt.Sprint(outgoingDataPacket, responseDataLength)   //push data length
				outgoingDataPacket = fmt.Sprint(outgoingDataPacket, incomingDataProtocol) //push protocol no.
				outgoingDataPacket = fmt.Sprint(outgoingDataPacket, serialNo)             //push serial no
				/* generate and push error code */
				data_p := fmt.Sprint(responseDataLength, incomingDataProtocol, serialNo)
				responseErrorHex, _ := hex.DecodeString(data_p)
				responseDataCRC := Checksum(responseErrorHex, table)                     //Error code in uint16
				outgoingDataErrorCode := strconv.FormatUint(uint64(responseDataCRC), 16) //Error code in string
				outgoingDataPacket = fmt.Sprint(outgoingDataPacket, outgoingDataErrorCode)

				outgoingDataPacket = fmt.Sprint(outgoingDataPacket, stopBits) //push stop bit
				/* send response to terminal */
				hexDataPacket, responseDataError := hex.DecodeString(outgoingDataPacket)
				conn.Write(hexDataPacket)
				/* set login status */
				if incomingDataProtocol == "01" && responseDataError == nil {
					loginState = true
				} else {
					fmt.Println("Response Data Error for :" + terminalId)
					fmt.Println(responseDataError.Error())
				}
			}
		} else {
			fmt.Println("*** UNKNOWN DATA: " + incomingDataPacket)
		}
	}
}

func timeDifferenceInMinutes(startTime time.Time, endTime time.Time) float64 {
	duration := endTime.Sub(startTime)
	diffInMinutes := duration.Minutes()
	return diffInMinutes
}

func hex2Datetime(hexStr string) string {
	dateTimeStrFromHex := "20"
	yearFromHex, converErr := hex2Int(hexStr[:2])
	monthFromHex, converErr := hex2Int(hexStr[2:4])
	dayFromHex, converErr := hex2Int(hexStr[4:6])
	hourFromHex, converErr := hex2Int(hexStr[6:8])
	minuteFromHex, converErr := hex2Int(hexStr[8:10])
	secondFromHex, converErr := hex2Int(hexStr[10:12])

	if converErr != nil {
		return ""
	}
	dateTimeStrFromHex = fmt.Sprint(dateTimeStrFromHex, yearFromHex, "-", monthFromHex, "-", dayFromHex, " ", hourFromHex, ":", minuteFromHex, ":", secondFromHex)
	return dateTimeStrFromHex
}

func hex2Int(hexStr string) (int64, error) {
	intValue, err := strconv.ParseInt(hexStr, 16, 0)
	if err != nil {
		return 0, err
	}
	return intValue, nil
}

func bin2Int(binStr string) (int64, error) {
	intValue, err := strconv.ParseInt(binStr, 2, 64)
	if err != nil {
		return 0, err
	}
	return intValue, nil
}

func hex2Bin(hexStr string) (string, error) {
	ui, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return "", err
	}

	format := fmt.Sprintf("%%0%db", len(hexStr)*4)
	return fmt.Sprintf(format, ui), nil
}

func ReverseByte(val byte) byte {
	var rval byte = 0
	for i := uint(0); i < 8; i++ {
		if val&(1<<i) != 0 {
			rval |= 0x80 >> i
		}
	}
	return rval
}

func ReverseUint8(val uint8) uint8 {
	return ReverseByte(val)
}

func ReverseUint16(val uint16) uint16 {
	var rval uint16 = 0
	for i := uint(0); i < 16; i++ {
		if val&(uint16(1)<<i) != 0 {
			rval |= uint16(0x8000) >> i
		}
	}
	return rval
}

// Params represents parameters of CRC-16 algorithms.
// More information about algorithms parametrization and parameter descriptions
// can be found here - http://www.zlib.net/crc_v3.txt
type Params struct {
	Poly   uint16
	Init   uint16
	RefIn  bool
	RefOut bool
	XorOut uint16
	Check  uint16
	Name   string
}

// Predefined CRC-16 algorithms.
// List of algorithms with their parameters borrowed from here -  http://reveng.sourceforge.net/crc-catalogue/16.htm
//
// The variables can be used to create Table for the selected algorithm.
var (
	CRC16_ARC         = Params{0x8005, 0x0000, true, true, 0x0000, 0xBB3D, "CRC-16/ARC"}
	CRC16_AUG_CCITT   = Params{0x1021, 0x1D0F, false, false, 0x0000, 0xE5CC, "CRC-16/AUG-CCITT"}
	CRC16_BUYPASS     = Params{0x8005, 0x0000, false, false, 0x0000, 0xFEE8, "CRC-16/BUYPASS"}
	CRC16_CCITT_FALSE = Params{0x1021, 0xFFFF, false, false, 0x0000, 0x29B1, "CRC-16/CCITT-FALSE"}
	CRC16_CDMA2000    = Params{0xC867, 0xFFFF, false, false, 0x0000, 0x4C06, "CRC-16/CDMA2000"}
	CRC16_DDS_110     = Params{0x8005, 0x800D, false, false, 0x0000, 0x9ECF, "CRC-16/DDS-110"}
	CRC16_DECT_R      = Params{0x0589, 0x0000, false, false, 0x0001, 0x007E, "CRC-16/DECT-R"}
	CRC16_DECT_X      = Params{0x0589, 0x0000, false, false, 0x0000, 0x007F, "CRC-16/DECT-X"}
	CRC16_DNP         = Params{0x3D65, 0x0000, true, true, 0xFFFF, 0xEA82, "CRC-16/DNP"}
	CRC16_EN_13757    = Params{0x3D65, 0x0000, false, false, 0xFFFF, 0xC2B7, "CRC-16/EN-13757"}
	CRC16_GENIBUS     = Params{0x1021, 0xFFFF, false, false, 0xFFFF, 0xD64E, "CRC-16/GENIBUS"}
	CRC16_MAXIM       = Params{0x8005, 0x0000, true, true, 0xFFFF, 0x44C2, "CRC-16/MAXIM"}
	CRC16_MCRF4XX     = Params{0x1021, 0xFFFF, true, true, 0x0000, 0x6F91, "CRC-16/MCRF4XX"}
	CRC16_RIELLO      = Params{0x1021, 0xB2AA, true, true, 0x0000, 0x63D0, "CRC-16/RIELLO"}
	CRC16_T10_DIF     = Params{0x8BB7, 0x0000, false, false, 0x0000, 0xD0DB, "CRC-16/T10-DIF"}
	CRC16_TELEDISK    = Params{0xA097, 0x0000, false, false, 0x0000, 0x0FB3, "CRC-16/TELEDISK"}
	CRC16_TMS37157    = Params{0x1021, 0x89EC, true, true, 0x0000, 0x26B1, "CRC-16/TMS37157"}
	CRC16_USB         = Params{0x8005, 0xFFFF, true, true, 0xFFFF, 0xB4C8, "CRC-16/USB"}
	CRC16_CRC_A       = Params{0x1021, 0xC6C6, true, true, 0x0000, 0xBF05, "CRC-16/CRC-A"}
	CRC16_KERMIT      = Params{0x1021, 0x0000, true, true, 0x0000, 0x2189, "CRC-16/KERMIT"}
	CRC16_MODBUS      = Params{0x8005, 0xFFFF, true, true, 0x0000, 0x4B37, "CRC-16/MODBUS"}
	CRC16_X_25        = Params{0x1021, 0xFFFF, true, true, 0xFFFF, 0x906E, "CRC-16/X-25"}
	CRC16_XMODEM      = Params{0x1021, 0x0000, false, false, 0x0000, 0x31C3, "CRC-16/XMODEM"}
)

// Table is a 256-word table representing polinomial and algorithm settings for efficient processing.
type Table struct {
	params Params
	data   [256]uint16
}

// MakeTable returns the Table constructed from the specified algorithm.
func MakeTable(params Params) *Table {
	table := new(Table)
	table.params = params
	for n := 0; n < 256; n++ {
		crc := uint16(n) << 8
		for i := 0; i < 8; i++ {
			bit := (crc & 0x8000) != 0
			crc <<= 1
			if bit {
				crc ^= params.Poly
			}
		}
		table.data[n] = crc
	}
	return table
}

// Init returns the initial value for CRC register corresponding to the specified algorithm.
func Init(table *Table) uint16 {
	return table.params.Init
}

// Update returns the result of adding the bytes in data to the crc.
func Update(crc uint16, data []byte, table *Table) uint16 {
	for _, d := range data {
		if table.params.RefIn {
			d = ReverseByte(d)
		}
		crc = crc<<8 ^ table.data[byte(crc>>8)^d]
	}
	return crc
}

// Complete returns the result of CRC calculation and post-calculation processing of the crc.
func Complete(crc uint16, table *Table) uint16 {
	if table.params.RefOut {
		return ReverseUint16(crc) ^ table.params.XorOut
	}
	return crc ^ table.params.XorOut
}

// Checksum returns CRC checksum of data usign scpecified algorithm represented by the Table.
func Checksum(data []byte, table *Table) uint16 {
	crc := Init(table)
	crc = Update(crc, data, table)
	return Complete(crc, table)
}
