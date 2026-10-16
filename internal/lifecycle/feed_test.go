package lifecycle_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

// fakeEoXAPI serves the token endpoint and EOXByProductID the way Cisco's API
// does: the IDs comma-separated in the path, one record per page so the
// client has to follow PaginationResponseRecord.
type fakeEoXAPI struct {
	requests atomic.Int64
	mu       sync.Mutex
	batches  [][]string
}

func (f *fakeEoXAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests.Add(1)
	if r.URL.Path == "/token" {
		body, _ := io.ReadAll(r.Body)
		form := string(body)
		if !strings.Contains(form, "client_id=id") || !strings.Contains(form, "client_secret=s3cret") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"tok","token_type":"Bearer"}`)
		return
	}
	if r.Header.Get("Authorization") != "Bearer tok" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/eox/EOXByProductID/"), "/")
	index, _ := strconv.Atoi(parts[0])
	ids := strings.Split(parts[1], ",")
	if index == 1 {
		f.mu.Lock()
		f.batches = append(f.batches, ids)
		f.mu.Unlock()
	}
	id := ids[index-1]
	record := fmt.Sprintf(`{"EOLProductID":%q,"EndOfSaleDate":{"value":"2016-10-30"}}`, id)
	if strings.HasPrefix(id, "UNKNOWN") {
		record = fmt.Sprintf(`{"EOLProductID":%q,"EOXError":{"ErrorID":"SSA_ERR_026"}}`, id)
	}
	_, _ = fmt.Fprintf(w, `{"PaginationResponseRecord":{"PageIndex":%d,"LastIndex":%d},"EOXRecord":[%s]}`,
		index, len(ids), record)
}

func newFakeAPI(t *testing.T) (*fakeEoXAPI, *httptest.Server, lifecycle.Credentials) {
	t.Helper()
	api := &fakeEoXAPI{}
	srv := httptest.NewTLSServer(api)
	t.Cleanup(srv.Close)
	return api, srv, lifecycle.Credentials{
		Endpoint: srv.URL + "/eox", TokenURL: srv.URL + "/token", ClientID: "id", ClientSecret: "s3cret",
	}
}

func TestFetcherFetch(t *testing.T) {
	api, srv, creds := newFakeAPI(t)
	ids := make([]string, 0, 25)
	for i := range 24 {
		ids = append(ids, fmt.Sprintf("WS-C%04d", i))
	}
	ids = append(ids, "UNKNOWN-1")

	table, err := lifecycle.NewFetcher(srv.Client()).Fetch(context.Background(), creds, ids)
	require.NoError(t, err)

	assert.Equal(t, 24, table.Len(), "the placeholder for an unknown ID is skipped")
	rec, ok := table.Lookup("WS-C0023")
	require.True(t, ok)
	assert.Equal(t, "2016-10-30", rec.EndOfSale)
	require.Len(t, api.batches, 2, "25 IDs go out as a batch of 20 and a batch of 5")
	assert.Len(t, api.batches[0], 20)
	assert.Len(t, api.batches[1], 5)
}

func TestFetcherFetchErrors(t *testing.T) {
	_, srv, creds := newFakeAPI(t)
	creds.ClientSecret = "wrong"
	_, err := lifecycle.NewFetcher(srv.Client()).Fetch(context.Background(), creds, []string{"WS-C1"})
	require.ErrorContains(t, err, "token request: HTTP 401")
	assert.NotContains(t, err.Error(), "wrong", "an error never carries the secret")

	notEoX := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"something":"else"}`)
	}))
	defer notEoX.Close()
	_, err = lifecycle.NewFetcher(notEoX.Client()).Fetch(context.Background(),
		lifecycle.Credentials{Endpoint: notEoX.URL}, []string{"WS-C1"})
	require.ErrorIs(t, err, lifecycle.ErrNotEoX)
}

func TestFeedUpdateValidate(t *testing.T) {
	tests := []struct {
		name      string
		update    lifecycle.FeedUpdate
		hasSecret bool
		wantErr   string
	}{
		{name: "answer off", update: lifecycle.FeedUpdate{Choice: outbound.ChoiceOff}},
		{
			name: "on with Cisco credentials",
			update: lifecycle.FeedUpdate{
				Choice: outbound.ChoiceOn, ClientID: "id", ClientSecret: "s",
			},
		},
		{
			name:      "on keeping the stored secret",
			update:    lifecycle.FeedUpdate{Choice: outbound.ChoiceOn, ClientID: "id"},
			hasSecret: true,
		},
		{
			name:   "on against a mirror with no credentials",
			update: lifecycle.FeedUpdate{Choice: outbound.ChoiceOn, Endpoint: "https://eox.example.net/v5"},
		},
		{
			name:    "cannot un-answer",
			update:  lifecycle.FeedUpdate{Choice: outbound.ChoiceUnanswered},
			wantErr: `choice must be "on" or "off"`,
		},
		{
			name:    "on against Cisco with no credentials",
			update:  lifecycle.FeedUpdate{Choice: outbound.ChoiceOn},
			wantErr: "needs a client ID and secret",
		},
		{
			name: "plain http endpoint",
			update: lifecycle.FeedUpdate{
				Choice: outbound.ChoiceOff, Endpoint: "http://eox.example.net",
			},
			wantErr: "endpoint: must be https",
		},
		{
			name: "credentials in the token URL",
			update: lifecycle.FeedUpdate{
				Choice: outbound.ChoiceOff, TokenURL: "https://u:p@id.example.net/token",
			},
			wantErr: "tokenUrl: must not carry credentials",
		},
		{
			name:    "client ID without a secret",
			update:  lifecycle.FeedUpdate{Choice: outbound.ChoiceOff, ClientID: "id"},
			wantErr: "a client ID needs a client secret",
		},
		{
			name: "clearing the secret leaves the ID without one",
			update: lifecycle.FeedUpdate{
				Choice: outbound.ChoiceOff, ClientID: "id", ClearClientSecret: true,
			},
			hasSecret: true,
			wantErr:   "a client ID needs a client secret",
		},
		{
			name: "secret and clear together",
			update: lifecycle.FeedUpdate{
				Choice: outbound.ChoiceOff, ClientID: "id", ClientSecret: "s", ClearClientSecret: true,
			},
			wantErr: "exclusive",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.update.Validate(tc.hasSecret)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, lifecycle.ErrInvalidFeed)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestTableMerge(t *testing.T) {
	stored, err := lifecycle.New([]lifecycle.EoXRecord{
		{ProductID: "WS-A", EndOfSale: "2016-01-01"},
		{ProductID: "WS-B", EndOfSale: "2017-01-01"},
	})
	require.NoError(t, err)
	fetched, err := lifecycle.New([]lifecycle.EoXRecord{
		{ProductID: "ws-b", EndOfSale: "2018-01-01"},
		{ProductID: "WS-C", EndOfSale: "2019-01-01"},
	})
	require.NoError(t, err)

	merged, err := stored.Merge(fetched)
	require.NoError(t, err)
	assert.Equal(t, []lifecycle.EoXRecord{
		{ProductID: "WS-A", EndOfSale: "2016-01-01"},
		{ProductID: "WS-C", EndOfSale: "2019-01-01"},
		{ProductID: "ws-b", EndOfSale: "2018-01-01"},
	}, merged.Records())
}

// memStore is an in-memory lifecycle.Store.
type memStore struct {
	mu    sync.Mutex
	table *lifecycle.Table
}

func (m *memStore) EoX(context.Context) (*lifecycle.Table, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.table == nil {
		return lifecycle.New(nil)
	}
	return m.table, nil
}

func (m *memStore) SetEoX(_ context.Context, table *lifecycle.Table) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.table = table
	return nil
}

func (m *memStore) MergeEoX(ctx context.Context, newer *lifecycle.Table) error {
	stored, err := m.EoX(ctx)
	if err != nil {
		return err
	}
	merged, err := stored.Merge(newer)
	if err != nil {
		return err
	}
	return m.SetEoX(ctx, merged)
}

// switchSource is a FeedSource whose answer a test flips.
type switchSource struct {
	on    atomic.Bool
	asked atomic.Int64
	creds lifecycle.Credentials
}

func (s *switchSource) Credentials(context.Context) (lifecycle.Credentials, bool, error) {
	s.asked.Add(1)
	return s.creds, s.on.Load(), nil
}

// TestRefresherOffMakesNoRequest is the acceptance's zero-outbound half: with
// the feed unanswered or off, start and wakes reach nothing. Turned on, the
// next wake fetches and merges into what was imported.
func TestRefresherOffMakesNoRequest(t *testing.T) {
	api, srv, creds := newFakeAPI(t)
	source := &switchSource{creds: creds}
	imported, err := lifecycle.New([]lifecycle.EoXRecord{{ProductID: "WS-IMPORTED"}})
	require.NoError(t, err)
	store := &memStore{table: imported}
	refresher := lifecycle.NewRefresher(source, func() []string { return []string{"WS-C1", "WS-C2"} },
		store, lifecycle.NewFetcher(srv.Client()), time.Hour, slog.New(slog.DiscardHandler))

	require.NoError(t, refresher.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, refresher.Stop(context.Background())) })
	// The loop consults the source at start and again after a wake; three
	// consultations mean two full off-passes ran after the first.
	for source.asked.Load() < 3 {
		refresher.Wake()
		runtime.Gosched()
	}
	assert.Zero(t, api.requests.Load(), "a feed that is not on makes no request")
	assert.True(t, refresher.Status().LastAttempt.IsZero())

	source.on.Store(true)
	refresher.Wake()
	require.Eventually(t, func() bool { return !refresher.Status().LastSuccess.IsZero() },
		5*time.Second, 10*time.Millisecond)
	status := refresher.Status()
	assert.Empty(t, status.LastError)
	assert.Equal(t, 2, status.Fetched)
	table, err := store.EoX(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, table.Len(), "fetched records join the imported one")
}

func TestRefresherRecordsFailure(t *testing.T) {
	_, srv, creds := newFakeAPI(t)
	creds.ClientSecret = "wrong"
	source := &switchSource{creds: creds}
	source.on.Store(true)
	refresher := lifecycle.NewRefresher(source, func() []string { return []string{"WS-C1"} },
		&memStore{}, lifecycle.NewFetcher(srv.Client()), time.Hour, slog.New(slog.DiscardHandler))

	require.NoError(t, refresher.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, refresher.Stop(context.Background())) })
	require.Eventually(t, func() bool { return !refresher.Status().LastAttempt.IsZero() },
		5*time.Second, 10*time.Millisecond)
	status := refresher.Status()
	assert.Contains(t, status.LastError, "HTTP 401")
	assert.True(t, status.LastSuccess.IsZero())
}
