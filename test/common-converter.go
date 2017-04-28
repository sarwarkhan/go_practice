package main

import (
	"fmt"
	"math"
	"strconv"
)

func main() {
	//hex string to int
	hTi := hex2int("0123456789012345")
	// integer value as string
	hTdecimalStr := strconv.Itoa(hTi)
	fmt.Printf("%T, %v\n", hTdecimalStr, hTdecimalStr)
	//fmt.Println(hTdecimalStr)

	// hex string to bin
	hTb := hex2bin("F8")
	fmt.Printf("%T, %v\n", hTb, hTb)

	//hex string to float
	hTf := hex2float("422a0000")
	//fmt.Println(f)
	fmt.Printf("%T, %v\n", hTf, hTf)

	//checkErr(err)
}

func bin(i int, prefix bool) string {
	i64 := int64(i)

	if prefix {
		return "0b" + strconv.FormatInt(i64, 2) // base 2 for binary
	} else {
		return strconv.FormatInt(i64, 2) // base 2 for binary
	}
}

func bin2int(binStr string) int {

	// base 2 for binary
	result, _ := strconv.ParseInt(binStr, 2, 64)
	return int(result)
}

func oct(i int, prefix bool) string {
	i64 := int64(i)

	if prefix {
		return "0o" + strconv.FormatInt(i64, 8) // base 8 for octal
	} else {
		return strconv.FormatInt(i64, 8) // base 8 for octal
	}
}

func oct2int(octStr string) int {
	// base 8 for octal
	result, _ := strconv.ParseInt(octStr, 8, 64)
	return int(result)
}

func hex(i int, prefix bool) string {
	i64 := int64(i)

	if prefix {
		return "0x" + strconv.FormatInt(i64, 16) // base 16 for hexadecimal
	} else {
		return strconv.FormatInt(i64, 16) // base 16 for hexadecimal
	}
}

func hex2int(hexStr string) int {
	// base 16 for hexadecimal
	result, _ := strconv.ParseInt(hexStr, 16, 64)
	return int(result)
}

func hex2bin(hexStr string) string {

	result, _ := strconv.ParseUint(hexStr, 16, 64)

	return fmt.Sprintf("%08b", result)
}

func hex2float(hexStr string) string {
	temp, _ := strconv.ParseUint(hexStr, 16, 64)
	result := math.Float64frombits(temp)
	return fmt.Sprintf("%.2f", result)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
