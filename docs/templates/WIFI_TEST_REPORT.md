# WiFi Adapter Test Report Template

Use this template to document WiFi adapter testing results. Submit via the
[Hardware Report](https://github.com/krisarmstrong/luminetiq/issues/new?template=hardware-report.yml)
issue template.

## Hardware Information

**Adapter**: [Make and Model] **Chipset**: [e.g., Intel AX200, Atheros AR9271] **Form Factor**:
[PCIe / USB / M.2] **Bus**: [PCI/PCIe / USB 2.0 / USB 3.0]

**Vendor ID/Product ID**: [lspci or lsusb output]

```
# For PCI devices
lspci -nn | grep -i network

# For USB devices
lsusb | grep -i wireless
```

## System Information

**Operating System**: [e.g., Ubuntu 22.04 LTS] **Kernel**: [uname -r] **Architecture**: [x86_64 /
arm64] **Driver**: [e.g., iwlwifi, ath9k_htc] **Driver Version**: [from ethtool -i or modinfo]

## Test Results

### Test Script Output

Run the hardware compatibility test script and paste the output:

```bash
sudo ./scripts/test-hardware-compatibility.sh wlan0
```

<details>
<summary>Full Test Output (click to expand)</summary>

```
[Paste full test script output here]
```

</details>

### Feature Matrix

Mark each feature as: ✅ Working | ⚠️ Partial | ❌ Not Working | ⏭️ Not Tested

| Feature                    | Status   | Notes                        |
| -------------------------- | -------- | ---------------------------- |
| **Interface Detection**    | ✅/⚠️/❌ | Detected as wlan0/wlp3s0/etc |
| **nl80211 Support**        | ✅/⚠️/❌ | `iw` command compatibility   |
| **Monitor Mode**           | ✅/⚠️/❌ | Packet injection capability  |
| **2.4 GHz Scanning**       | ✅/⚠️/❌ | Channels 1-11/13             |
| **5 GHz Scanning**         | ✅/⚠️/❌ | Channels 36-165              |
| **6 GHz Scanning**         | ✅/⚠️/❌ | WiFi 6E only                 |
| **Signal Strength (RSSI)** | ✅/⚠️/❌ | Accuracy of readings         |
| **Channel Switching**      | ✅/⚠️/❌ | Set specific channels        |
| **Managed Mode**           | ✅/⚠️/❌ | Normal client operation      |
| **Packet Capture**         | ✅/⚠️/❌ | tcpdump/Wireshark            |

### Performance Notes

**Scan Performance**:

- Networks found: [count]
- Scan duration: [seconds]
- Signal range: [min dBm] to [max dBm]

**Stability**:

- Interface switching: [Stable / Occasional drops / Unstable]
- Mode changes: [Reliable / Requires restart / Fails]
- Long-term operation: [Hours tested without issues]

**Special Configuration**:

```bash
# Any special setup required
# e.g., firmware loading, module parameters, etc.
```

## LuminetIQ Feature Compatibility

Test with actual LuminetIQ features:

### WiFi Site Survey

- [ ] Floor plan upload works
- [ ] Sample collection works
- [ ] Signal strength accurate
- [ ] Heatmap generation correct

### Network Discovery

- [ ] Detects nearby networks
- [ ] Reports SSID correctly
- [ ] Reports BSSID correctly
- [ ] Reports security correctly

### Interface Switching

- [ ] Can switch from/to this interface
- [ ] Settings persist after switch
- [ ] No crashes during switch

## Issues Encountered

Document any problems:

**Issue 1**: [Description]

- Severity: Critical / Major / Minor
- Workaround: [If any]
- Error messages: [Paste relevant logs]

**Issue 2**: [Description]

- ...

## Recommendation

**Overall Assessment**: Excellent / Good / Fair / Poor / Not Recommended

**Use Case Recommendations**:

- ✅ **Site Surveys**: [Yes/No - reason]
- ✅ **Network Discovery**: [Yes/No - reason]
- ✅ **Development/Testing**: [Yes/No - reason]
- ✅ **Production Deployment**: [Yes/No - reason]

**Summary**: [1-2 sentence recommendation]

## Additional Notes

[Any other observations, tips, or recommendations for future users]

---

**Tested By**: [Your GitHub username] **Test Date**: [YYYY-MM-DD] **Script Version**: [Git commit
hash or script version]
