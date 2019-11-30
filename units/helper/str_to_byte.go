package helper

import "fmt"

// from C++ std library

func StrToBytes(str string) uint64 {
	res := uint64(1)
	pos := len(str) - 1
	if pos > 0 && (str[pos] == 'b' || str[pos] == 'B') {
		pos--
	}
	if pos > 0 {
		switch str[pos] {
		case 'p':
			fallthrough
		case 'P':
			res *= 1024
			fallthrough
		case 't':
			fallthrough
		case 'T':
			res *= 1024
			fallthrough
		case 'g':
			fallthrough
		case 'G':
			res *= 1024
			fallthrough
		case 'm':
			fallthrough
		case 'M':
			res *= 1024
			fallthrough
		case 'k':
			fallthrough
		case 'K':
			res *= 1024
		}
	}
	if res == 1 {
		// read as long long
		res = toUint64(str)
	} else {
		// read as double so that the user can use things like 0.5mb
		res *= uint64(toFloat64(str))
	}
	return res
}

func toUint64(str string) uint64 {
	var res uint64
	_, err := fmt.Sscanf(str, "%d", &res)
	if err != nil {
		return 0
	}
	return res
}

func toFloat64(str string) float64 {
	var res float64
	_, err := fmt.Sscanf(str, "%g", &res)
	if err != nil {
		return 0
	}
	return res
}

func BytesToStr() {

}
