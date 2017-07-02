require('events').EventEmitter.defaultMaxListeners = 0;

var net = require('net');
var CRC16 = require('./utility/crc-itu.js');
var CONVERT = require('./utility/converter.js');
var TIMEDIFF = require('./utility/time-difference.js');
var DB = require('./config/database.js');

var PORT = 6969;

var server = net.createServer();
server.listen(PORT);
console.log('Server listening on ' + ': '+ PORT);
server.on('connection', function(sock) {
    sock.setEncoding('hex');
    // client connected
    console.log('CONNECTED: ' + sock.remoteAddress +':'+ sock.remotePort);
    var loginState = false;
    var locationDataFlag = false;
    var locationDataLastTime = new Date();
    var terminalId = '';
    /* keeping sensor data record -start  */
    var engineStatus = 0;
    var fuelConnectionStatus = '0';
    var gpsTrackingStatus = '1';
    var alarmStatus = '00';
    var alarmType = '000';
    var chargeStatus = '0';
    var defenceStatus = '0';
    var voltageLevelStatus = '04';
    var gsmSignalStrength = '03';
    var alarmLanguage = '01';
    var acStatus = 0;
    var movingDirection = '';
    /* keeping sensor data record -end */

    /* Keeping location data record -start */
    var dateTimeFormated = '';
    var dateFormated = '';
    var latitudeInDecimalMinutes = '';
    var LongitudeInDecimalMinutes = '';
    var speedInDecimal = '0.00';
    var e_w_indicator = 'E';
    var n_s_indicator = 'N';
    var bearing = '0';
    /* Keeping location data record -end */

    sock.on('data', function(data) {

        var initialBits = data.substr(0,4);
        if(initialBits == '7878') { // normal data
          /*convert time to GMT+06:00*/
          var systemDate = new Date();
          var deviceTZO = 360;
          var date = new Date(systemDate.getTime() + (60000*(systemDate.getTimezoneOffset()+deviceTZO)));
          //console.log('Date-Time : '+ date.getFullYear() +'-'+ (date.getMonth() + 1)+'-'+date.getDate()+' '+date.getHours()+':'+date.getMinutes()+':'+date.getSeconds());

          //assume valid data
          var dataType = 'A';

          //data sent from terminal split by stop bits
          var incomingDataArray = data.split('0d0a');
          // loop through the data
          for(var z=0;z<incomingDataArray.length-1;z++) {
            console.log(' ----- Chunk data processing Start -----');
            var incomingDataPacket = incomingDataArray[z]+'0d0a';
            console.log(z+'# Input Data : '+incomingDataPacket);
            //incoming data packet length
            var incomingDataPacketLength = parseInt(incomingDataPacket.length);
            //incoming data start bit
            var startBits = incomingDataPacket.substr(0,4);

            //incoming data length
            var incomingDataLength = incomingDataPacket.substr(4,2);

            //incoming data protocol
            var incomingDataProtocol = incomingDataPacket.substr(6,2);

            //Serial No from incoming data
            var serialNoPosition = incomingDataPacketLength - 12;
            var serialNo = incomingDataPacket.substr(serialNoPosition, 4);

            //Error code from incoming data
            var errorCodeStartPosition = incomingDataPacketLength - 8;
            var incomingDataErrorCode = incomingDataPacket.substr(errorCodeStartPosition, 4)

            //stop bit
            var stopBitsPosition = incomingDataPacketLength - 4;
            var stopBits = incomingDataPacket.substr(stopBitsPosition, 4);
            //var stopBits = '0D0A';

            //error code checking string
            var errorCodeCheckStrLength = incomingDataPacketLength - 4 - 8;
            var strForErrorCode = incomingDataPacket.substr(4, errorCodeCheckStrLength);

            // check error code
            let crcCheck = CRC16.checkErrorCode(strForErrorCode);

            if(incomingDataErrorCode == crcCheck) {
                dataType = 'A';
            } else {
                dataType = 'V';
                console.log(':: INVALID DATA :: DATE-TIME > '+ date.getFullYear() +'-'+ (date.getMonth() + 1)+'-'+date.getDate()+' '+date.getHours()+':'+date.getMinutes()+':'+date.getSeconds());
                // console.log('Incoming data: '+incomingDataPacket);
                // console.log('CRC DATA: '+strForErrorCode);
                // console.log(incomingDataProtocol+' Calculated CRC : '+crcCheck);
            }

            //Handle Login data Terminal id
            if(incomingDataProtocol == '01' && dataType == 'A') {
                terminalId = incomingDataPacket.substr(8,16);
            }
            console.log('____ TERMINAL ID :'+terminalId);

            //handle heart-bit data
            if(incomingDataProtocol == '13' && loginState == true && dataType == 'A') {
                let terminalStatus = incomingDataPacket.substr(8, 2);
                voltageLevelStatus = incomingDataPacket.substr(10, 2);
                gsmSignalStrength = incomingDataPacket.substr(12, 2);
                alarmStatus = incomingDataPacket.substr(14, 2);
                alarmLanguage = incomingDataPacket.substr(16, 2);

                //convert terminal information into binary
                let sensorDataBinary = CONVERT.hex2bin(terminalStatus);
                //Update global variable
                fuelConnectionStatus = sensorDataBinary.substr(0, 1);
                gpsTrackingStatus = sensorDataBinary.substr(1, 1);
                alarmType = sensorDataBinary.substr(2, 3);
                chargeStatus = sensorDataBinary.substr(5, 1);
                //Update global engineStatus
                engineStatus = parseInt(sensorDataBinary.substr(6, 1));
                if(engineStatus == 0) {
                    speedInDecimal = '0.00';
                }
                defenceStatus = sensorDataBinary.substr(7, 1);

                //prepare insert query for gps_data table
                if(locationDataFlag == true) {
                    let lastLocationDataTimeDiff = TIMEDIFF.diffInSecond(locationDataLastTime, new Date());
                    if(lastLocationDataTimeDiff >= 180) {
                      // set current time
                      dateTimeFormated = date.getFullYear()+'-'+(date.getMonth() + 1)+'-'+date.getDate()+' '+date.getHours()+':'+date.getMinutes()+':'+date.getSeconds();
                      dateFormated = date.getFullYear()+'-'+(date.getMonth() + 1)+'-'+date.getDate();
                      console.log('---- heart-bit data inserted ----');
                      let insertSql = "INSERT INTO `gps_data_tr06` (device_emei, record_date, record_time, data_status, engine_status, speed, latitude, longitude, n_s_indicator, e_w_indicator, bearing, direction, ac_status, fuel_connection_status, gps_tracking_status, alarm_status, alarm_type, charge_status, defence_status, voltage_level, gsm_signal_strength, alarm_language) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";
                      let params = [terminalId, dateFormated, dateTimeFormated, dataType, engineStatus, speedInDecimal, latitudeInDecimalMinutes, LongitudeInDecimalMinutes, n_s_indicator, e_w_indicator, bearing, movingDirection, acStatus, fuelConnectionStatus, gpsTrackingStatus, alarmStatus, alarmType, chargeStatus, defenceStatus, voltageLevelStatus, gsmSignalStrength, alarmLanguage];
                      // insert into sensor_data_curr table
                      DB.query(insertSql, params, function(data, error){
                          if(error) {
                            console.log('*** DB ERROR : ' + error);
                          } else {
                            //console.log(data);
                          }
                      });
                      //update previous location data update time
                      locationDataLastTime = new Date();
                    }
                }

            }

            //For location data
            if(incomingDataProtocol == '12' && loginState == true) {
                let hexDatetime = incomingDataPacket.substr(8, 12);
                let quantityOfGPS = incomingDataPacket.substr(20, 2);
                let hexLatitude = incomingDataPacket.substr(22, 8);
                let hexLongitude = incomingDataPacket.substr(30, 8);
                let hexSpeed = incomingDataPacket.substr(38, 2);
                let hexCourseStatus = incomingDataPacket.substr(40, 4);
                let hexMCC = incomingDataPacket.substr(44, 4);
                let hexMNC = incomingDataPacket.substr(48, 2);
                let hexLAC = incomingDataPacket.substr(50, 4);
                let hexCellID = incomingDataPacket.substr(54, 6);

                //location data datetime
                dateTimeFormated = CONVERT.hex2datetime(hexDatetime);
                console.log('*** LOCATION DATA TIME : '+dateTimeFormated);
                let dateFormatedArray = dateTimeFormated.split(' ');
                dateFormated = dateFormatedArray[0];
                //Calculate latitude
                let latitudeDecimal = CONVERT.hex2dec(hexLatitude);
                latitudeInDecimalMinutes = (latitudeDecimal/30000)/60;
                //Calculate Longitude
                let longitudeDecimal = CONVERT.hex2dec(hexLongitude);
                LongitudeInDecimalMinutes = (longitudeDecimal/30000)/60;
                //Calculate speed in km/h
                speedInDecimal = CONVERT.hex2Float(hexSpeed);
                //Calculate course and status
                let byte_1 = hexCourseStatus.substr(0, 2);
                let byte_2 = hexCourseStatus.substr(2, 2);
                //convert to binary
                let byteBinary = CONVERT.hex2bin(byte_1)+CONVERT.hex2bin(byte_2);
                e_w_indicator = (byteBinary.substr(4, 1) == 0) ? 'E' : 'W';
                n_s_indicator = (byteBinary.substr(5, 1) == 0) ? 'S' : 'N';
                // determine moving direction;
                if(n_s_indicator == 'N') {
                  movingDirection = 'north';
                } else {
                  movingDirection = 'south';
                }
                if(e_w_indicator == 'E') {
                  movingDirection += '-east';
                } else {
                  movingDirection += '-west';
                }
                bearing = CONVERT.bin2dec(byteBinary.substr(6, 10)); //course in decimal degree

                //prepare insert query for gps_data_tr06 table
                let insertSql = "INSERT INTO `gps_data_tr06` (device_emei, record_date, record_time, data_status, engine_status, speed, latitude, longitude, n_s_indicator, e_w_indicator, bearing, direction, ac_status, fuel_connection_status, gps_tracking_status, alarm_status, alarm_type, charge_status, defence_status, voltage_level, gsm_signal_strength, alarm_language) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";
                let params = [terminalId, dateFormated, dateTimeFormated, dataType, engineStatus, speedInDecimal, latitudeInDecimalMinutes, LongitudeInDecimalMinutes, n_s_indicator, e_w_indicator, bearing, movingDirection, acStatus, fuelConnectionStatus, gpsTrackingStatus, alarmStatus, alarmType, chargeStatus, defenceStatus, voltageLevelStatus, gsmSignalStrength, alarmLanguage];
                // insert into sensor_data_tr06 table
                DB.query(insertSql, params, function(data, error){
                  if(error) {
                    console.log('*** DB ERROR : ' + error);
                  } else {
                    //console.log(data);
                  }
                });
                //set locationDataFlag to true
                locationDataFlag = true;
                locationDataLastTime = new Date();

            }

            //handle alarm data
            if(incomingDataProtocol == '16' && loginState == true) {
                let hexDatetime = incomingDataPacket.substr(8, 12);
                let quantityOfGPS = incomingDataPacket.substr(20, 2);
                let hexLatitude = incomingDataPacket.substr(22, 8);
                let hexLongitude = incomingDataPacket.substr(30, 8);
                let hexSpeed = incomingDataPacket.substr(38, 2);
                let hexCourseStatus = incomingDataPacket.substr(40, 4);
                let hexLBSLength = incomingDataPacket.substr(44, 2);
                let hexMCC = incomingDataPacket.substr(46, 4);
                let hexMNC = incomingDataPacket.substr(50, 2);
                let hexLAC = incomingDataPacket.substr(52, 4);
                let hexCellID = incomingDataPacket.substr(56, 6);

                //alarm data datetime
                dateTimeFormated = CONVERT.hex2datetime(hexDatetime);
                console.log('*** ALARM DATA TIME : '+dateTimeFormated);
                let dateFormatedArray = dateTimeFormated.split(' ');
                dateFormated = dateFormatedArray[0];
                //Calculate latitude
                let latitudeDecimal = CONVERT.hex2dec(hexLatitude);
                latitudeInDecimalMinutes = (latitudeDecimal/30000)/60;
                //Calculate Longitude
                let longitudeDecimal = CONVERT.hex2dec(hexLongitude);
                LongitudeInDecimalMinutes = (longitudeDecimal/30000)/60;
                //Calculate speed in km/h
                speedInDecimal = CONVERT.hex2Float(hexSpeed);
                //Calculate course and status
                let byte_1 = hexCourseStatus.substr(0, 2);
                let byte_2 = hexCourseStatus.substr(2, 2);
                //convert to binary
                let byteBinary = CONVERT.hex2bin(byte_1)+CONVERT.hex2bin(byte_2);
                e_w_indicator = (byteBinary.substr(4, 1) == 0) ? 'E' : 'W';
                n_s_indicator = (byteBinary.substr(5, 1) == 0) ? 'S' : 'N';
                // determine moving direction;
                if(n_s_indicator == 'N') {
                  movingDirection = 'north';
                } else {
                  movingDirection = 'south';
                }
                if(e_w_indicator == 'E') {
                  movingDirection += '-east';
                } else {
                  movingDirection += '-west';
                }
                bearing = CONVERT.bin2dec(byteBinary.substr(6, 10)); //course in decimal degree
                //sensore data
                let terminalStatus = incomingDataPacket.substr(62, 2);
                voltageLevelStatus = incomingDataPacket.substr(64, 2);
                gsmSignalStrength = incomingDataPacket.substr(66, 2);
                alarmStatus = incomingDataPacket.substr(68, 2);
                alarmLanguage = incomingDataPacket.substr(70, 2);

                //convert terminal information into binary
                let sensorDataBinary = CONVERT.hex2bin(terminalStatus);
                //Update global variables
                fuelConnectionStatus = sensorDataBinary.substr(0, 1);
                gpsTrackingStatus = sensorDataBinary.substr(1, 1);
                alarmType = sensorDataBinary.substr(2, 3);
                chargeStatus = sensorDataBinary.substr(5, 1);
                //Update global engineStatus
                engineStatus = sensorDataBinary.substr(6, 1);
                defenceStatus = sensorDataBinary.substr(7, 1);

                //prepare insert query for gps_data table
                let insertSql = "INSERT INTO `gps_data_tr06` (device_emei, record_date, record_time, data_status, engine_status, speed, latitude, longitude, n_s_indicator, e_w_indicator, bearing, direction, ac_status, fuel_connection_status, gps_tracking_status, alarm_status, alarm_type, charge_status, defence_status, voltage_level, gsm_signal_strength, alarm_language) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";
                let params = [terminalId, dateFormated, dateTimeFormated, dataType, engineStatus, speedInDecimal, latitudeInDecimalMinutes, LongitudeInDecimalMinutes, n_s_indicator, e_w_indicator, bearing, movingDirection, acStatus, fuelConnectionStatus, gpsTrackingStatus, alarmStatus, alarmType, chargeStatus, defenceStatus, voltageLevelStatus, gsmSignalStrength, alarmLanguage];
                // insert into sensor_data_curr table
                DB.query(insertSql, params, function(data, error){
                  if(error) {
                    console.log('*** DB ERROR : ' + error);
                  } else {
                    //console.log(data);
                  }
                });
                //set locationDataFlag to true
                locationDataFlag = true;
                locationDataLastTime = new Date();

            }

            // write back to client incase of login/heart-bit data
            if(dataType == 'A' && (incomingDataProtocol == '01' || incomingDataProtocol == '13')) {
                //prepare response data
                let outgoingDataPacket = startBits; // initialize with start bits

                //push data length
                let responseDataLength = CONVERT.dec2hex(5);
                outgoingDataPacket = outgoingDataPacket + responseDataLength;

                //push protocol no.
                outgoingDataPacket = outgoingDataPacket + incomingDataProtocol;

                //push serial no
                outgoingDataPacket = outgoingDataPacket + serialNo;

                //Error code
                data_p = responseDataLength+incomingDataProtocol+serialNo;
                outgoingDataErrorCode = CRC16.checkErrorCode(data_p);
                outgoingDataPacket = outgoingDataPacket + outgoingDataErrorCode;

                //push stop bit
                outgoingDataPacket = outgoingDataPacket + stopBits;

                // send response to terminal
                try {
                  let hexDataPacket = Buffer.from(outgoingDataPacket, 'hex');
                  sock.write(hexDataPacket);
                  /* for local */
                  // sock.write(outgoingDataPacket);

                  //set login status
                  if(incomingDataProtocol == '01') {
                      loginState = true;
                  }
                } catch (e) {
                    console.log(e);
                }
            }
            console.log(' ----- Chunk data processing End -----');
          }

        } else {
            console.log('*** UNKNOWN DATA: '+data);
        }
    });

});
