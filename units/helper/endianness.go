package helper

func IsLittleEndian() bool {
	s := uint16(0xAAFF)
	b := uint8(s)
	return b == 0xFF
}

func IsBigEndian() bool {
	return !IsLittleEndian()
}
