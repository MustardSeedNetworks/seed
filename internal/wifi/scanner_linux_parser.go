//go:build linux

package wifi

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ht40Width         = 40
	vht80Width        = 80
	vht160Width       = 160
	defaultNoiseFloor = -95
)

type iwScanPatterns struct {
	bss, frequency, signal, ssid *regexp.Regexp
	ht, vht, he, rsn, wpa, wep   *regexp.Regexp
}

type iwScanState struct {
	patterns iwScanPatterns
	networks []*ScannedNetwork
	current  *ScannedNetwork
	inRSN    bool
	inWPA    bool
}

func newIWScanState() *iwScanState {
	return &iwScanState{patterns: iwScanPatterns{
		bss:       regexp.MustCompile(`^BSS ([0-9a-fA-F:]{17})`),
		frequency: regexp.MustCompile(`freq:\s*(\d+)`),
		signal:    regexp.MustCompile(`signal:\s*(-?\d+(?:\.\d+)?)\s*dBm`),
		ssid:      regexp.MustCompile(`SSID:\s*(.*)`),
		ht:        regexp.MustCompile(`\* secondary channel offset: (above|below|no secondary)`),
		vht:       regexp.MustCompile(`\* channel width: (\d+)\s*\((\d+)\)?\s*MHz`),
		he:        regexp.MustCompile(`HE capabilities`),
		rsn:       regexp.MustCompile(`RSN:`),
		wpa:       regexp.MustCompile(`WPA:`),
		wep:       regexp.MustCompile(`WEP:`),
	}}
}

func (state *iwScanState) consume(line string) {
	if match := state.patterns.bss.FindStringSubmatch(line); match != nil {
		state.finishCurrent()
		state.current = &ScannedNetwork{
			BSSID:        strings.ToUpper(match[1]),
			LastSeen:     time.Now(),
			ChannelWidth: defaultChannelWidth,
			NoiseFloor:   defaultNoiseFloor,
		}
		state.inRSN, state.inWPA = false, false
		return
	}
	if state.current == nil {
		return
	}
	trimmed := strings.TrimSpace(line)
	state.parseRadio(trimmed)
	state.parseWidth(trimmed)
	state.parseSecurity(trimmed)
}

func (state *iwScanState) parseRadio(line string) {
	if match := state.patterns.frequency.FindStringSubmatch(line); match != nil {
		frequency, _ := strconv.Atoi(match[1])
		state.current.Frequency = frequency
		state.current.Channel = frequencyToChannel(frequency)
		state.current.IsDFS = isDFSFrequency(frequency)
	}
	if match := state.patterns.signal.FindStringSubmatch(line); match != nil {
		signal, _ := strconv.ParseFloat(match[1], 64)
		state.current.Signal = int(signal)
		state.current.SNR = state.current.Signal - state.current.NoiseFloor
	}
	if match := state.patterns.ssid.FindStringSubmatch(line); match != nil {
		state.current.SSID = match[1]
	}
}

func isDFSFrequency(frequency int) bool {
	const (
		lowerDFSStart = 5250
		lowerDFSEnd   = 5350
		upperDFSStart = 5470
		upperDFSEnd   = 5725
	)
	return (frequency >= lowerDFSStart && frequency <= lowerDFSEnd) ||
		(frequency >= upperDFSStart && frequency <= upperDFSEnd)
}

func (state *iwScanState) parseWidth(line string) {
	if state.patterns.ht.MatchString(line) {
		if strings.Contains(line, "above") || strings.Contains(line, "below") {
			state.current.ChannelWidth, state.current.HTMode = ht40Width, "HT40"
		} else {
			state.current.HTMode = "HT20"
		}
	}
	if match := state.patterns.vht.FindStringSubmatch(line); match != nil {
		state.setVHTWidth(match[1])
	}
	heModeAllowed := state.current.HTMode == "" || strings.HasPrefix(state.current.HTMode, "HT")
	if state.patterns.he.MatchString(line) && heModeAllowed {
		state.current.HTMode = "HE" + strconv.Itoa(state.current.ChannelWidth)
	}
}

func (state *iwScanState) setVHTWidth(value string) {
	width, _ := strconv.Atoi(value)
	if width > state.current.ChannelWidth {
		state.current.ChannelWidth = width
	}
	switch width {
	case vht80Width:
		state.current.HTMode = "VHT80"
	case vht160Width:
		state.current.HTMode = "VHT160"
	}
}

func (state *iwScanState) parseSecurity(line string) {
	if state.patterns.rsn.MatchString(line) {
		state.inRSN, state.inWPA = true, false
	}
	if state.patterns.wpa.MatchString(line) {
		state.inWPA, state.inRSN = true, false
	}
	if state.inRSN || state.inWPA {
		state.setSecurity(line)
	}
	if state.patterns.wep.MatchString(line) && state.current.Security == "" {
		state.current.Security = "WEP"
	}
	if strings.Contains(line, "capability:") && !strings.Contains(line, "Privacy") && state.current.Security == "" {
		state.current.Security = "Open"
	}
}

func (state *iwScanState) setSecurity(line string) {
	switch {
	case strings.Contains(line, "SAE"):
		state.current.Security = "WPA3"
	case strings.Contains(line, "PSK") && state.current.Security != "WPA3" && state.inRSN:
		state.current.Security = "WPA2"
	case strings.Contains(line, "PSK") && state.current.Security != "WPA3":
		state.current.Security = "WPA"
	case strings.Contains(line, "802.1X") || strings.Contains(line, "EAP"):
		state.current.Security = "WPA2-Enterprise"
	}
}

func (state *iwScanState) finishCurrent() {
	if state.current == nil || state.current.BSSID == "" {
		return
	}
	if state.current.Security == "" {
		state.current.Security = "Unknown"
	}
	if state.current.HTMode == "" && state.current.ChannelWidth >= ht40Width {
		state.current.HTMode = fmt.Sprintf("HT%d", state.current.ChannelWidth)
	}
	state.networks = append(state.networks, state.current)
}

func (state *iwScanState) result() []*ScannedNetwork {
	state.finishCurrent()
	return state.networks
}
