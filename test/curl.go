package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	lat, lng := "-33.8670", "151.1957"

	location, err := getLocationAddress(lat, lng)
	checkErr(err)
	fmt.Println(location)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func getLocationAddress(lat string, lng string) (string, error) {
	formatedAddress := ""
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%s,%s", lat, lng)
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
