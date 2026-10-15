// Package lifecycle tracks vendor end-of-life milestones for the hardware
// Seed has inventoried (P-D9): when a product stops being sold, stops getting
// software fixes, stops getting security fixes, and stops being supported.
//
// Seed ships no lifecycle data. The operator imports Cisco's EoX data as a
// file, in the shape the Cisco EoX API returns, or opts in to [Refresher]
// fetching it from that API. The feed is off until the operator turns it on,
// so an air-gapped install never reaches out because of this package.
package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// maxRecords bounds a table. It is operator input stored in the database;
// the whole Cisco EoX catalogue for an estate is a few hundred product IDs.
const maxRecords = 20000

// dateLayout is the form of every milestone date, the one the EoX API uses.
const dateLayout = "2006-01-02"

// Status is how far through its end-of-life a product is: the latest
// milestone whose date has passed.
type Status string

const (
	// StatusNotListed means the imported data has no record for the product.
	StatusNotListed Status = "not_listed"
	// StatusAnnounced means an end-of-life is announced and no milestone has
	// passed yet.
	StatusAnnounced Status = "announced"
	// StatusEndOfSale means the product can no longer be ordered.
	StatusEndOfSale Status = "end_of_sale"
	// StatusEndOfSWMaintenance means no further software fix releases.
	StatusEndOfSWMaintenance Status = "end_of_sw_maintenance"
	// StatusEndOfSecuritySupport means no further security fixes.
	StatusEndOfSecuritySupport Status = "end_of_security_support"
	// StatusEndOfSupport means the vendor supports the product no longer.
	StatusEndOfSupport Status = "end_of_support"
)

// EoXRecord is one product's end-of-life milestones. A date is "YYYY-MM-DD", or
// empty when the vendor has not announced it.
type EoXRecord struct {
	ProductID            string `json:"productId"`
	Description          string `json:"description,omitempty"`
	EndOfSale            string `json:"endOfSale,omitempty"`
	EndOfSWMaintenance   string `json:"endOfSwMaintenance,omitempty"`
	EndOfSecuritySupport string `json:"endOfSecuritySupport,omitempty"`
	LastDateOfSupport    string `json:"lastDateOfSupport,omitempty"`
	Bulletin             string `json:"bulletin,omitempty"`
	MigrationProductID   string `json:"migrationProductId,omitempty"`
}

// Status reports the latest milestone that has passed on day.
func (r EoXRecord) Status(day time.Time) Status {
	today := day.Format(dateLayout)
	milestones := []struct {
		date   string
		status Status
	}{
		{r.LastDateOfSupport, StatusEndOfSupport},
		{r.EndOfSecuritySupport, StatusEndOfSecuritySupport},
		{r.EndOfSWMaintenance, StatusEndOfSWMaintenance},
		{r.EndOfSale, StatusEndOfSale},
	}
	for _, m := range milestones {
		// ISO dates order as strings.
		if m.date != "" && m.date <= today {
			return m.status
		}
	}
	return StatusAnnounced
}

// Table is a validated set of records keyed by product ID. It is immutable
// once built.
type Table struct {
	records []EoXRecord
	byID    map[string]int
}

// New validates records and builds a table. A product ID is matched without
// regard to case or surrounding space, and may appear once.
//
// Errors identify a record by position and never quote it: the data is
// operator input, and its text must not reach a log line verbatim.
func New(records []EoXRecord) (*Table, error) {
	if len(records) > maxRecords {
		return nil, fmt.Errorf("table has %d records, limit %d", len(records), maxRecords)
	}
	t := &Table{records: make([]EoXRecord, 0, len(records)), byID: make(map[string]int, len(records))}
	for i, rec := range records {
		key := normalizeID(rec.ProductID)
		if key == "" {
			return nil, fmt.Errorf("record %d has no product ID", i)
		}
		if _, dup := t.byID[key]; dup {
			return nil, fmt.Errorf("record %d repeats an earlier product ID", i)
		}
		for _, date := range []string{
			rec.EndOfSale, rec.EndOfSWMaintenance, rec.EndOfSecuritySupport, rec.LastDateOfSupport,
		} {
			if date == "" {
				continue
			}
			if _, err := time.Parse(dateLayout, date); err != nil {
				return nil, fmt.Errorf("record %d has a date that is not YYYY-MM-DD", i)
			}
		}
		if rec.Bulletin != "" && !strings.HasPrefix(rec.Bulletin, "https://") &&
			!strings.HasPrefix(rec.Bulletin, "http://") {
			return nil, fmt.Errorf("record %d has a bulletin link that is not http(s)", i)
		}
		rec.ProductID = strings.TrimSpace(rec.ProductID)
		t.byID[key] = len(t.records)
		t.records = append(t.records, rec)
	}
	return t, nil
}

// Len is the number of records.
func (t *Table) Len() int { return len(t.records) }

// Records returns the records in product ID order.
func (t *Table) Records() []EoXRecord {
	out := slices.Clone(t.records)
	slices.SortFunc(out, func(a, b EoXRecord) int { return strings.Compare(a.ProductID, b.ProductID) })
	return out
}

// Lookup returns the record for a product ID.
func (t *Table) Lookup(productID string) (EoXRecord, bool) {
	i, ok := t.byID[normalizeID(productID)]
	if !ok {
		return EoXRecord{}, false
	}
	return t.records[i], true
}

// Assess returns the record for a product ID and its status on day, or
// [StatusNotListed] when the table has no record for it.
func (t *Table) Assess(productID string, day time.Time) (EoXRecord, Status) {
	rec, ok := t.Lookup(productID)
	if !ok {
		return EoXRecord{}, StatusNotListed
	}
	return rec, rec.Status(day)
}

// MarshalJSON stores a table as its records.
func (t *Table) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Records())
}

// Parse reads a table stored by [Table.MarshalJSON].
func Parse(data []byte) (*Table, error) {
	var records []EoXRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode lifecycle table: %w", err)
	}
	return New(records)
}

func normalizeID(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

// ErrNotEoX is returned for a document that is not a Cisco EoX API response.
var ErrNotEoX = errors.New("not a Cisco EoX response: no EOXRecord array")

// ciscoResponse is the part of a Cisco EoX API v5 response Seed reads.
type ciscoResponse struct {
	EOXRecord  *[]ciscoRecord `json:"EOXRecord"`
	Pagination struct {
		PageIndex int `json:"PageIndex"`
		LastIndex int `json:"LastIndex"`
	} `json:"PaginationResponseRecord"`
}

type ciscoRecord struct {
	EOLProductID                string    `json:"EOLProductID"`
	ProductIDDescription        string    `json:"ProductIDDescription"`
	LinkToProductBulletinURL    string    `json:"LinkToProductBulletinURL"`
	EndOfSaleDate               ciscoDate `json:"EndOfSaleDate"`
	EndOfSWMaintenanceReleases  ciscoDate `json:"EndOfSWMaintenanceReleases"`
	EndOfSecurityVulSupportDate ciscoDate `json:"EndOfSecurityVulSupportDate"`
	LastDateOfSupport           ciscoDate `json:"LastDateOfSupport"`
	EOXMigrationDetails         struct {
		MigrationProductID string `json:"MigrationProductId"`
	} `json:"EOXMigrationDetails"`
	// EOXError is set on the placeholder the API returns for a product ID it
	// has no end-of-life record for.
	EOXError *struct{} `json:"EOXError"`
}

type ciscoDate struct {
	Value string `json:"value"`
}

// ParseCiscoEoX reads Cisco EoX API responses as the operator downloaded
// them: one response object, or an array of them for a query that spanned
// several pages or calls. Placeholders for product IDs Cisco has no record
// for are skipped; a product ID in more than one response is kept once.
func ParseCiscoEoX(data []byte) (*Table, error) {
	var pages []ciscoResponse
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &pages); err != nil {
			return nil, fmt.Errorf("decode EoX responses: %w", err)
		}
	} else {
		var page ciscoResponse
		if err := json.Unmarshal(trimmed, &page); err != nil {
			return nil, fmt.Errorf("decode EoX response: %w", err)
		}
		pages = []ciscoResponse{page}
	}

	return tableFromPages(pages)
}

// tableFromPages builds a table from EoX API responses, skipping placeholders
// and keeping a product ID in more than one response once.
func tableFromPages(pages []ciscoResponse) (*Table, error) {
	var records []EoXRecord
	seen := map[string]bool{}
	for _, page := range pages {
		if page.EOXRecord == nil {
			return nil, ErrNotEoX
		}
		for _, rec := range *page.EOXRecord {
			key := normalizeID(rec.EOLProductID)
			if rec.EOXError != nil || key == "" || seen[key] {
				continue
			}
			seen[key] = true
			records = append(records, EoXRecord{
				ProductID:            rec.EOLProductID,
				Description:          rec.ProductIDDescription,
				EndOfSale:            rec.EndOfSaleDate.Value,
				EndOfSWMaintenance:   rec.EndOfSWMaintenanceReleases.Value,
				EndOfSecuritySupport: rec.EndOfSecurityVulSupportDate.Value,
				LastDateOfSupport:    rec.LastDateOfSupport.Value,
				Bulletin:             rec.LinkToProductBulletinURL,
				MigrationProductID:   rec.EOXMigrationDetails.MigrationProductID,
			})
		}
	}
	return New(records)
}

// Merge returns a table holding t's records with newer's in place of any
// with the same product ID, so a fetch updates what it covered and keeps the
// rest of an imported file.
func (t *Table) Merge(newer *Table) (*Table, error) {
	records := slices.Clone(newer.records)
	for _, rec := range t.records {
		if _, replaced := newer.byID[normalizeID(rec.ProductID)]; !replaced {
			records = append(records, rec)
		}
	}
	return New(records)
}

// Store keeps the imported table. An install with nothing imported has an
// empty table.
type Store interface {
	EoX(ctx context.Context) (*Table, error)
	SetEoX(ctx context.Context, table *Table) error
	// MergeEoX replaces the stored records newer covers and keeps the rest,
	// atomically with respect to SetEoX.
	MergeEoX(ctx context.Context, newer *Table) error
}
