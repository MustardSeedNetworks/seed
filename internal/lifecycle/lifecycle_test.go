package lifecycle_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
)

func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return d
}

func TestParseCiscoEoX(t *testing.T) {
	data, err := os.ReadFile("testdata/eox-response.json")
	require.NoError(t, err)

	table, err := lifecycle.ParseCiscoEoX(data)
	require.NoError(t, err)

	// The placeholder for GLC-TE, which Cisco has no record for, is skipped.
	require.Equal(t, 2, table.Len())
	rec, ok := table.Lookup(" ws-c3750x-48p-s ")
	require.True(t, ok, "lookup ignores case and surrounding space")
	assert.Equal(t, lifecycle.EoXRecord{
		ProductID:            "WS-C3750X-48P-S",
		Description:          "Catalyst 3750X 48 Port PoE IP Base",
		EndOfSale:            "2016-10-30",
		EndOfSWMaintenance:   "2017-10-30",
		EndOfSecuritySupport: "2019-10-30",
		LastDateOfSupport:    "2021-10-31",
		Bulletin: "https://www.cisco.com/c/en/us/products/collateral/switches/" +
			"catalyst-3750-x-series-switches/eos-eol-notice-c51-736181.html",
		MigrationProductID: "WS-C3850-48P-S",
	}, rec)
	_, ok = table.Lookup("GLC-TE")
	assert.False(t, ok)
}

func TestParseCiscoEoXPages(t *testing.T) {
	page := func(pid string) string {
		return `{"EOXRecord":[{"EOLProductID":"` + pid + `","EndOfSaleDate":{"value":"2020-01-01"}}]}`
	}
	table, err := lifecycle.ParseCiscoEoX([]byte("[" + page("A-1") + "," + page("B-2") + "," + page("a-1") + "]"))
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len(), "a product ID in two pages is kept once")

	empty, err := lifecycle.ParseCiscoEoX([]byte(`{"EOXRecord":[]}`))
	require.NoError(t, err)
	assert.Zero(t, empty.Len(), "a response with no records clears the table")
}

func TestParseCiscoEoXRejects(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"not JSON", `EOLProductID,EndOfSaleDate`},
		{"not an EoX response", `{"records":[]}`},
		{"a page that is not an EoX response", `[{"EOXRecord":[]},{}]`},
		{"bad date", `{"EOXRecord":[{"EOLProductID":"A-1","EndOfSaleDate":{"value":"30/10/2016"}}]}`},
		{"script bulletin", `{"EOXRecord":[{"EOLProductID":"A-1","LinkToProductBulletinURL":"javascript:alert(1)"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := lifecycle.ParseCiscoEoX([]byte(tt.data))
			assert.Error(t, err)
		})
	}
}

func TestNewRejects(t *testing.T) {
	_, err := lifecycle.New([]lifecycle.EoXRecord{{ProductID: " "}})
	require.Error(t, err)
	_, err = lifecycle.New([]lifecycle.EoXRecord{{ProductID: "A-1"}, {ProductID: "a-1"}})
	require.Error(t, err)
	_, err = lifecycle.New(make([]lifecycle.EoXRecord, 20001))
	require.Error(t, err)
}

func TestRecordStatus(t *testing.T) {
	rec := lifecycle.EoXRecord{
		ProductID:            "A-1",
		EndOfSale:            "2016-10-30",
		EndOfSWMaintenance:   "2017-10-30",
		EndOfSecuritySupport: "2019-10-30",
		LastDateOfSupport:    "2021-10-31",
	}
	tests := []struct {
		day  string
		want lifecycle.Status
	}{
		{"2016-10-29", lifecycle.StatusAnnounced},
		{"2016-10-30", lifecycle.StatusEndOfSale},
		{"2018-01-01", lifecycle.StatusEndOfSWMaintenance},
		{"2019-10-30", lifecycle.StatusEndOfSecuritySupport},
		{"2026-10-15", lifecycle.StatusEndOfSupport},
	}
	for _, tt := range tests {
		t.Run(tt.day, func(t *testing.T) {
			assert.Equal(t, tt.want, rec.Status(day(t, tt.day)))
		})
	}

	// An unannounced milestone is skipped, not treated as passed.
	partial := lifecycle.EoXRecord{ProductID: "B-2", EndOfSale: "2026-09-01", LastDateOfSupport: "2031-09-30"}
	assert.Equal(t, lifecycle.StatusEndOfSale, partial.Status(day(t, "2029-01-01")))
}

func TestAssess(t *testing.T) {
	table, err := lifecycle.New([]lifecycle.EoXRecord{{ProductID: "A-1", EndOfSale: "2020-01-01"}})
	require.NoError(t, err)

	rec, status := table.Assess("a-1", day(t, "2026-10-15"))
	assert.Equal(t, lifecycle.StatusEndOfSale, status)
	assert.Equal(t, "A-1", rec.ProductID)

	_, status = table.Assess("Z-9", day(t, "2026-10-15"))
	assert.Equal(t, lifecycle.StatusNotListed, status)
}

func TestStoredRoundTrip(t *testing.T) {
	table, err := lifecycle.New([]lifecycle.EoXRecord{
		{ProductID: "B-2", LastDateOfSupport: "2031-09-30"},
		{ProductID: "A-1", EndOfSale: "2020-01-01", Bulletin: "https://example.com/eol"},
	})
	require.NoError(t, err)
	data, err := json.Marshal(table)
	require.NoError(t, err)

	back, err := lifecycle.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, table.Records(), back.Records())
	assert.Equal(t, "A-1", back.Records()[0].ProductID, "records come back in product ID order")
}
