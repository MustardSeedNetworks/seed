//go:build linux

package wifi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/wifi"
	"golang.org/x/sys/unix"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
)

// scanTimeout bounds one trigger-and-dump; a scan across 2.4 and 5 GHz takes a
// few seconds, a stuck wireless stack must not hang the caller.
const scanTimeout = 30 * time.Second

const (
	mBmPerDBm            = 100
	macLen               = 6
	capabilityPrivacyBit = 0x0010 // 802.11 Capability Information: Privacy
	signalPercentMaximum = 100
	signalDbmRange       = 70
	signalDbmMinimum     = -100
)

// scanPlatform triggers an nl80211 scan and reads the kernel's BSS table with
// each entry's raw information elements, which dot11 decodes into the fields
// the Wi-Fi anomaly rules read (ADR-0030: the nmcli and iw text it replaces
// carried none of them beyond the security suite).
func scanPlatform(iface string, _ Helper) ([]*ScannedNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	client, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("open nl80211: %w", err)
	}
	defer client.Close()

	ifi, err := wirelessInterface(client, iface)
	if err != nil {
		return nil, err
	}

	// Triggering needs CAP_NET_ADMIN, and fails with EBUSY while another scan
	// (usually wpa_supplicant's) runs. The kernel table still holds the most
	// recent scan's results in both cases, so read those.
	if scanErr := client.Scan(ctx, ifi); scanErr != nil {
		if !errors.Is(scanErr, unix.EPERM) && !errors.Is(scanErr, unix.EBUSY) {
			return nil, fmt.Errorf("trigger scan on %s: %w", iface, scanErr)
		}
		logging.GetLogger().Debug("wifi scan not triggered; reading the last results",
			"interface", iface, "error", scanErr)
	}

	msgs, err := dumpScan(ctx, ifi.Index)
	if err != nil {
		return nil, fmt.Errorf("read scan results on %s: %w", iface, err)
	}
	networks, err := parseScanDump(msgs)
	if err != nil {
		return nil, fmt.Errorf("parse scan results on %s: %w", iface, err)
	}

	// Survey noise is per channel and most drivers report it only for the one
	// the radio is tuned to; elsewhere NoiseFloor stays 0, never a guess.
	if surveys, surveyErr := client.SurveyInfo(ifi); surveyErr == nil {
		applySurveyNoise(networks, surveys)
	}
	return networks, nil
}

func wirelessInterface(client *wifi.Client, name string) (*wifi.Interface, error) {
	interfaces, err := client.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list wireless interfaces: %w", err)
	}
	for _, ifi := range interfaces {
		if ifi.Name == name {
			return ifi, nil
		}
	}
	return nil, fmt.Errorf("%s is not a wireless interface", name)
}

// dumpScan issues NL80211_CMD_GET_SCAN itself because mdlayher/wifi's
// AccessPoints keeps only the SSID, BSS Load and RSN elements.
func dumpScan(ctx context.Context, ifindex int) ([]genetlink.Message, error) {
	conn, err := genetlink.Dial(nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if deadlineErr := conn.SetDeadline(deadline); deadlineErr != nil {
			return nil, deadlineErr
		}
	}

	family, err := conn.GetFamily(unix.NL80211_GENL_NAME)
	if err != nil {
		return nil, err
	}
	if ifindex <= 0 || ifindex > math.MaxInt32 {
		return nil, fmt.Errorf("interface index %d out of range", ifindex)
	}
	ae := netlink.NewAttributeEncoder()
	ae.Uint32(unix.NL80211_ATTR_IFINDEX, uint32(ifindex))
	data, err := ae.Encode()
	if err != nil {
		return nil, err
	}
	return conn.Execute(genetlink.Message{
		Header: genetlink.Header{Command: unix.NL80211_CMD_GET_SCAN, Version: family.Version},
		Data:   data,
	}, family.ID, netlink.Request|netlink.Dump)
}

// scanEntry is one NL80211_ATTR_BSS as the kernel reports it.
type scanEntry struct {
	bssid      net.HardwareAddr
	freqMHz    int
	signalMBm  int
	hasMBm     bool
	signalPct  int
	capability uint16
	ies        []byte
	beaconIEs  []byte
}

func parseScanDump(msgs []genetlink.Message) ([]*ScannedNetwork, error) {
	networks := make([]*ScannedNetwork, 0, len(msgs))
	for _, m := range msgs {
		ad, err := netlink.NewAttributeDecoder(m.Data)
		if err != nil {
			return nil, err
		}
		for ad.Next() {
			if ad.Type() != unix.NL80211_ATTR_BSS {
				continue
			}
			var e scanEntry
			ad.Nested(e.decode)
			if len(e.bssid) == macLen {
				networks = append(networks, e.network())
			}
		}
		if decodeErr := ad.Err(); decodeErr != nil {
			return nil, decodeErr
		}
	}
	return networks, nil
}

func (e *scanEntry) decode(ad *netlink.AttributeDecoder) error {
	for ad.Next() {
		switch ad.Type() {
		case unix.NL80211_BSS_BSSID:
			e.bssid = net.HardwareAddr(ad.Bytes())
		case unix.NL80211_BSS_FREQUENCY:
			e.freqMHz = int(ad.Uint32())
		case unix.NL80211_BSS_SIGNAL_MBM:
			e.signalMBm = int(ad.Int32())
			e.hasMBm = true
		case unix.NL80211_BSS_SIGNAL_UNSPEC:
			e.signalPct = int(ad.Uint8())
		case unix.NL80211_BSS_CAPABILITY:
			e.capability = ad.Uint16()
		case unix.NL80211_BSS_INFORMATION_ELEMENTS:
			e.ies = ad.Bytes()
		case unix.NL80211_BSS_BEACON_IES:
			e.beaconIEs = ad.Bytes()
		}
	}
	return nil
}

func (e *scanEntry) network() *ScannedNetwork {
	ies := e.ies
	if len(ies) == 0 {
		ies = e.beaconIEs
	}
	bss := dot11.DecodeIEs(ies, e.capability&capabilityPrivacyBit != 0, e.freqMHz)

	signal := e.signalMBm / mBmPerDBm
	if !e.hasMBm {
		// Only a few full-MAC drivers report an unspecified 0-100 figure.
		signal = percentToDbm(e.signalPct)
	}

	return &ScannedNetwork{
		SSID:               bss.SSID,
		BSSID:              strings.ToUpper(e.bssid.String()),
		Signal:             signal,
		Channel:            bss.ChannelNum,
		Frequency:          e.freqMHz,
		Security:           bss.Security.String(),
		ChannelWidth:       bss.ChannelWidthM,
		HTMode:             htModeFor(bss.Standard, bss.ChannelWidthM),
		IsDFS:              isDFSFrequency(e.freqMHz),
		LastSeen:           time.Now(),
		Hidden:             bss.Hidden,
		Standard:           bss.Standard.String(),
		CountryCode:        bss.CountryCode,
		PMFRequired:        bss.PMFRequired,
		WPSEnabled:         bss.WPSEnabled,
		RRMNeighbor:        bss.RRMNeighbor,
		BTMSupported:       bss.BTMSupported,
		FTSupported:        bss.FTSupported,
		HasBSSLoad:         bss.HasBSSLoad,
		ChannelUtil:        bss.ChannelUtilByte,
		AdvertisedStations: bss.StationCount,
	}
}

// htModeFor names the PHY and width in the vocabulary detectChannelWidth and
// the discovery bridge parse (HT40, VHT80, HE20, EHT320). Pre-HT PHYs have none.
func htModeFor(standard dot11.Standard, width int) string {
	var phy string
	switch standard {
	case dot11.Standard80211n:
		phy = "HT"
	case dot11.Standard80211ac:
		phy = "VHT"
	case dot11.Standard80211ax:
		phy = "HE"
	case dot11.Standard80211be, dot11.Standard80211bn:
		phy = "EHT"
	case dot11.StandardUnknown, dot11.Standard80211a, dot11.Standard80211b, dot11.Standard80211g:
		return ""
	}
	return fmt.Sprintf("%s%d", phy, width)
}

// isDFSFrequency reports the 5 GHz U-NII-2A and U-NII-2C ranges, where an AP
// must vacate a channel on detecting radar.
func isDFSFrequency(freq int) bool {
	return (freq >= 5250 && freq <= 5350) || (freq >= 5470 && freq <= 5725)
}

func applySurveyNoise(networks []*ScannedNetwork, surveys []*wifi.SurveyInfo) {
	noise := make(map[int]int, len(surveys))
	for _, s := range surveys {
		if s.Noise != 0 {
			noise[s.Frequency] = s.Noise
		}
	}
	for _, n := range networks {
		if floor, ok := noise[n.Frequency]; ok {
			n.NoiseFloor = floor
			n.SNR = n.Signal - floor
		}
	}
}

// percentToDbm maps a 0-100 signal figure linearly onto -100..-30 dBm.
func percentToDbm(percent int) int {
	return signalDbmMinimum + (percent * signalDbmRange / signalPercentMaximum)
}
