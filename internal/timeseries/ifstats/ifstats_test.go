package ifstats_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
)

type fakeStore struct {
	ifaces []ifstats.Interface
	err    error
}

func (f fakeStore) Interfaces(context.Context) ([]ifstats.Interface, error) {
	return f.ifaces, f.err
}

func rated(target string, ifIndex uint32, inErrors, outErrors float64) ifstats.Interface {
	return ifstats.Interface{
		TargetID: target, TargetName: target, IfIndex: ifIndex,
		Rates: &ifstats.Rates{InErrors: inErrors, OutErrors: outErrors},
	}
}

func unrated(target string, ifIndex uint32) ifstats.Interface {
	return ifstats.Interface{TargetID: target, TargetName: target, IfIndex: ifIndex}
}

func TestServiceListOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []ifstats.Interface
		want []string
	}{
		{
			name: "highest error rate in both directions first",
			in:   []ifstats.Interface{rated("a", 1, 1, 0), rated("a", 2, 0, 5), rated("a", 3, 2, 2)},
			want: []string{"a/2", "a/3", "a/1"},
		},
		{
			name: "no rate yet after every rated interface, even an error-free one",
			in:   []ifstats.Interface{unrated("a", 1), rated("a", 2, 0, 0), unrated("b", 1)},
			want: []string{"a/2", "a/1", "b/1"},
		},
		{
			name: "ties by target then ifIndex",
			in:   []ifstats.Interface{rated("b", 1, 1, 0), rated("a", 10, 1, 0), rated("a", 2, 1, 0)},
			want: []string{"a/2", "a/10", "b/1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ifstats.NewService(fakeStore{ifaces: tc.in}).List(context.Background())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d interfaces, want %d", len(got), len(tc.want))
			}
			for i, iface := range got {
				if key := iface.TargetID + "/" + strconv.FormatUint(uint64(iface.IfIndex), 10); key != tc.want[i] {
					t.Errorf("position %d = %s, want %s", i, key, tc.want[i])
				}
			}
		})
	}
}

func TestServiceListPassesStoreError(t *testing.T) {
	t.Parallel()
	_, err := ifstats.NewService(fakeStore{err: ifstats.ErrUnavailable}).List(context.Background())
	if !errors.Is(err, ifstats.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}

func TestOperStatusFromMIB(t *testing.T) {
	t.Parallel()
	tests := map[int]ifstats.OperStatus{
		0: ifstats.OperUnknown, 1: ifstats.OperUp, 2: ifstats.OperDown, 3: ifstats.OperTesting,
		4: ifstats.OperUnknown, 5: ifstats.OperDormant, 6: ifstats.OperNotPresent,
		7: ifstats.OperLowerLayerDown, 8: ifstats.OperUnknown,
	}
	for in, want := range tests {
		if got := ifstats.OperStatusFromMIB(in); got != want {
			t.Errorf("OperStatusFromMIB(%d) = %s, want %s", in, got, want)
		}
	}
}
