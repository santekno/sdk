package timex_test

import (
	"testing"

	"github.com/santekno/sdk/timex"
)

func FuzzParseID(f *testing.F) {
	f.Add(timex.LongID, "Sabtu, 17 Mei 2026")
	f.Add(timex.DateID, "17-05-2026")
	f.Add(timex.ShortID, "17/05/26")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, layout, value string) {
		_, _ = timex.ParseID(layout, value)
	})
}
