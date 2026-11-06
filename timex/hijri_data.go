package timex

// The Umm al-Qura table below is copied from UMALQURA_MONTHLENGTH in ICU
// (icu4c/source/i18n/islamcal.cpp, tag release-76-1, Unicode-3.0 licence), the
// table behind the JavaScript Intl calendar "islamic-umalqura", restricted to
// the years 1356-1500 H. java.time's "islamic-umalqura" variant carries the
// same month lengths.
//
// Provenance matters here: Umm al-Qura data is not uniform across publishers.
// ICU agrees month for month with R.H. van Gent's reconstruction of the KACST
// calendar only for 1420-1450 H (17 April 1999 to 13 May 2029). Outside that
// window the two differ on 695 of the 1740 month starts, next at 1 Rabi II
// 1451 H (ICU 12 August 2029, van Gent 11 August 2029). The hijridate Python
// package follows van Gent, and .NET matches van Gent from 1402 H on. This
// table keeps the ICU values on purpose, so that results stay identical to
// Intl-based code; it is not a statement that ICU is the KACST-published
// calendar outside 1420-1450 H.
const (
	ummAlQuraFirstYear = 1356
	ummAlQuraLastYear  = 1500
	ummAlQuraMonths    = (ummAlQuraLastYear - ummAlQuraFirstYear + 1) * 12

	// ummAlQuraFirstDay is 1 Muharram 1356 H (14 March 1937) in days since
	// 1970-01-01.
	ummAlQuraFirstDay = -11981
)

// ummAlQuraMonthBits has one entry per Hijri year. Bit 11 is Muharram and bit 0
// is Dzulhijjah, as in the ICU source; a set bit is a 30-day month and a clear
// bit a 29-day month.
var ummAlQuraMonthBits = [ummAlQuraLastYear - ummAlQuraFirstYear + 1]uint16{
	0x0557, 0x0297, 0x054B, 0x06A3, 0x0752, // 1356-1360
	0x0B65, 0x056A, 0x0AAB, 0x052B, 0x0C95, // 1361-1365
	0x0D4A, 0x0DA5, 0x05CA, 0x0AD6, 0x0957, // 1366-1370
	0x04AB, 0x094B, 0x0AA5, 0x0B52, 0x0B6A, // 1371-1375
	0x0575, 0x0276, 0x08B7, 0x045B, 0x0555, // 1376-1380
	0x05A9, 0x05B4, 0x09DA, 0x04DD, 0x026E, // 1381-1385
	0x0936, 0x0AAA, 0x0D54, 0x0DB2, 0x05D5, // 1386-1390
	0x02DA, 0x095B, 0x04AB, 0x0A55, 0x0B49, // 1391-1395
	0x0B64, 0x0B71, 0x05B4, 0x0AB5, 0x0A55, // 1396-1400
	0x0D25, 0x0E92, 0x0EC9, 0x06D4, 0x0AE9, // 1401-1405
	0x096B, 0x04AB, 0x0A93, 0x0D49, 0x0DA4, // 1406-1410
	0x0DB2, 0x0AB9, 0x04BA, 0x0A5B, 0x052B, // 1411-1415
	0x0A95, 0x0B2A, 0x0B55, 0x055C, 0x04BD, // 1416-1420
	0x023D, 0x091D, 0x0A95, 0x0B4A, 0x0B5A, // 1421-1425
	0x056D, 0x02B6, 0x093B, 0x049B, 0x0655, // 1426-1430
	0x06A9, 0x0754, 0x0B6A, 0x056C, 0x0AAD, // 1431-1435
	0x0555, 0x0B29, 0x0B92, 0x0BA9, 0x05D4, // 1436-1440
	0x0ADA, 0x055A, 0x0AAB, 0x0595, 0x0749, // 1441-1445
	0x0764, 0x0BAA, 0x05B5, 0x02B6, 0x0A56, // 1446-1450
	0x0E4D, 0x0B25, 0x0B52, 0x0B6A, 0x05AD, // 1451-1455
	0x02AE, 0x092F, 0x0497, 0x064B, 0x06A5, // 1456-1460
	0x06AC, 0x0AD6, 0x055D, 0x049D, 0x0A4D, // 1461-1465
	0x0D16, 0x0D95, 0x05AA, 0x05B5, 0x02DA, // 1466-1470
	0x095B, 0x04AD, 0x0595, 0x06CA, 0x06E4, // 1471-1475
	0x0AEA, 0x04F5, 0x02B6, 0x0956, 0x0AAA, // 1476-1480
	0x0B54, 0x0BD2, 0x05D9, 0x02EA, 0x096D, // 1481-1485
	0x04AD, 0x0A95, 0x0B4A, 0x0BA5, 0x05B2, // 1486-1490
	0x09B5, 0x04D6, 0x0A97, 0x0547, 0x0693, // 1491-1495
	0x0749, 0x0B55, 0x056A, 0x0A6B, 0x052B, // 1496-1500
}
