package validx

import (
	"strconv"
	"time"

	"github.com/santekno/sdk/errx"
	"github.com/santekno/sdk/timex"
)

// NIKInfo is the structured result of parsing an Indonesian National ID
// (Nomor Induk Kependudukan). NIKs are 16 digits encoding province, regency,
// birth date, gender, and a serial number.
type NIKInfo struct {
	Raw          string
	ProvinceCode string
	Province     string
	RegencyCode  string
	Regency      string
	BirthDate    time.Time
	Gender       string // "M" or "F"
	SerialNumber string
}

// ParseNIK validates and decodes a 16-digit NIK string.
//
// On success returns the structured *NIKInfo. On failure returns a
// *errx.AppError with one of these codes:
//
//   - NIK_INVALID_LENGTH
//   - NIK_NOT_NUMERIC
//   - NIK_INVALID_PROVINCE
//   - NIK_INVALID_DATE
func ParseNIK(nik string) (*NIKInfo, error) {
	if len(nik) != 16 {
		return nil, errx.New("NIK_INVALID_LENGTH", "NIK must be exactly 16 digits").
			WithDetail("length", len(nik))
	}
	for _, r := range nik {
		if r < '0' || r > '9' {
			return nil, errx.New("NIK_NOT_NUMERIC", "NIK must contain only digits")
		}
	}

	provinceCode := nik[0:2]
	provinceName, ok := provinces[provinceCode]
	if !ok {
		return nil, errx.New("NIK_INVALID_PROVINCE",
			"NIK province code is not in the BPS registry").
			WithDetail("province_code", provinceCode)
	}

	regencyCode := nik[0:4]

	// Birth date is digits 6..12 (DDMMYY). For females, day is offset by +40.
	dayRaw, _ := strconv.Atoi(nik[6:8])
	month, _ := strconv.Atoi(nik[8:10])
	yearTwoDigit, _ := strconv.Atoi(nik[10:12])

	gender := "M"
	day := dayRaw
	if dayRaw > 40 {
		gender = "F"
		day = dayRaw - 40
	}

	// Birth year: yearTwoDigit + 1900 if year > current2digit else +2000.
	currentYear := time.Now().Year()
	century := (currentYear / 100) * 100
	birthYear := century + yearTwoDigit
	if birthYear > currentYear {
		birthYear -= 100
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return nil, errx.New("NIK_INVALID_DATE",
			"NIK contains an invalid birth date").
			WithDetail("day", day).WithDetail("month", month).WithDetail("year", birthYear)
	}

	birthDate := time.Date(birthYear, time.Month(month), day, 0, 0, 0, 0, timex.Jakarta)
	if birthDate.Month() != time.Month(month) || birthDate.Day() != day {
		return nil, errx.New("NIK_INVALID_DATE",
			"NIK birth date is not a valid calendar date").
			WithDetail("day", day).WithDetail("month", month).WithDetail("year", birthYear)
	}

	return &NIKInfo{
		Raw:          nik,
		ProvinceCode: provinceCode,
		Province:     provinceName,
		RegencyCode:  regencyCode,
		Regency:      "", // regency lookup table not bundled at v0.1
		BirthDate:    birthDate,
		Gender:       gender,
		SerialNumber: nik[12:16],
	}, nil
}

// provinces maps the BPS 2-digit province code to its name.
// Subset of the official BPS registry (38 provinces as of 2026).
var provinces = map[string]string{
	"11": "Aceh",
	"12": "Sumatera Utara",
	"13": "Sumatera Barat",
	"14": "Riau",
	"15": "Jambi",
	"16": "Sumatera Selatan",
	"17": "Bengkulu",
	"18": "Lampung",
	"19": "Kepulauan Bangka Belitung",
	"21": "Kepulauan Riau",
	"31": "DKI Jakarta",
	"32": "Jawa Barat",
	"33": "Jawa Tengah",
	"34": "DI Yogyakarta",
	"35": "Jawa Timur",
	"36": "Banten",
	"51": "Bali",
	"52": "Nusa Tenggara Barat",
	"53": "Nusa Tenggara Timur",
	"61": "Kalimantan Barat",
	"62": "Kalimantan Tengah",
	"63": "Kalimantan Selatan",
	"64": "Kalimantan Timur",
	"65": "Kalimantan Utara",
	"71": "Sulawesi Utara",
	"72": "Sulawesi Tengah",
	"73": "Sulawesi Selatan",
	"74": "Sulawesi Tenggara",
	"75": "Gorontalo",
	"76": "Sulawesi Barat",
	"81": "Maluku",
	"82": "Maluku Utara",
	"91": "Papua Barat",
	"92": "Papua Barat Daya",
	"93": "Papua Selatan",
	"94": "Papua",
	"95": "Papua Tengah",
	"96": "Papua Pegunungan",
}
