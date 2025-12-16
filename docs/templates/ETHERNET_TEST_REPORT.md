# Ethernet NIC Test Report Template

Use this template to document Ethernet NIC testing results. Submit via the
[Hardware Report](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)
issue template.

## Hardware Information

**NIC**: [Make and Model] **Chipset**: [e.g., Intel I350, Realtek RTL8111] **Form Factor**: [PCIe /
USB / Onboard] **Ports**: [Number of ports and speeds, e.g., 4x 1GbE, 2x 10GbE]

**Vendor ID/Product ID**: [lspci or lsusb output]

```
# For PCI devices
lspci -nn | grep -i ethernet

# For USB devices
lsusb | grep -i ethernet
```

## System Information

**Operating System**: [e.g., Ubuntu 22.04 LTS] **Kernel**: [uname -r] **Architecture**: [x86_64 /
arm64] **Driver**: [e.g., igb, r8169, e1000e] **Driver Version**: [from ethtool -i or modinfo]
**Firmware Version**: [from ethtool -i]

## Test Results

### Test Script Output

Run the hardware compatibility test script and paste the output:

```bash
sudo ./scripts/test-hardware-compatibility.sh eth0
```

<details>
<summary>Full Test Output (click to expand)</summary>

```
[Paste full test script output here]
```

</details>

### Feature Matrix

Mark each feature as: ✅ Working | ⚠️ Partial | ❌ Not Working | ⏭️ Not Tested

| Feature                 | Status   | Notes                        |
| ----------------------- | -------- | ---------------------------- |
| **Interface Detection** | ✅/⚠️/❌ | Detected as eth0/enp3s0/etc  |
| **Link Detection**      | ✅/⚠️/❌ | Carrier detect working       |
| **Speed Detection**     | ✅/⚠️/❌ | 10/100/1000/10000 Mbps       |
| **Duplex Detection**    | ✅/⚠️/❌ | Half/Full duplex             |
| **Autonegotiation**     | ✅/⚠️/❌ | Auto speed/duplex            |
| **TDR Cable Testing**   | ✅/⚠️/❌ | **Critical for diagnostics** |
| **Cable Length**        | ✅/⚠️/❌ | Distance to fault            |
| **Cable Fault Type**    | ✅/⚠️/❌ | Open/Short/OK/Impedance      |
| **Statistics**          | ✅/⚠️/❌ | Packet counters accuracy     |
| **Offload Features**    | ✅/⚠️/❌ | TSO, GSO, etc.               |
| **Jumbo Frames**        | ✅/⚠️/❌ | MTU > 1500                   |
| **VLAN Support**        | ✅/⚠️/❌ | 802.1Q tagging               |

### TDR Cable Testing (Critical Feature)

**TDR Support**: ✅ Yes / ❌ No

If supported, document test results:

**Test Scenario 1**: Good Cable (Cat6, 10m)

```bash
sudo ethtool --cable-test eth0
```

- Result: [PASS/FAIL]
- Length reported: [meters]
- Pair status: [OK/Open/Short]
- Accuracy: [Excellent/Good/Poor]

**Test Scenario 2**: Cable with Fault (if available)

- Fault type: [Open/Short/Unknown]
- Expected distance: [meters]
- Reported distance: [meters]
- Detection: [Accurate/Approximate/Failed]

**Test Scenario 3**: No Cable (unplugged)

- Detection: [Correctly reports no link]
- Error handling: [Graceful/Error/Crash]

### Performance Notes

**Link Speed**:

- 10 Mbps: ✅/⚠️/❌/⏭️
- 100 Mbps: ✅/⚠️/❌/⏭️
- 1000 Mbps (1 GbE): ✅/⚠️/❌/⏭️
- 10000 Mbps (10 GbE): ✅/⚠️/❌/⏭️

**Stability**:

- Interface reliability: [Stable / Occasional drops / Unstable]
- Statistics accuracy: [Accurate / Minor drift / Inaccurate]
- Long-term operation: [Hours tested without issues]

**Special Configuration**:

```bash
# Any special setup required
# e.g., driver parameters, firmware updates, etc.
```

## The Seed Feature Compatibility

Test with actual The Seed features:

### Cable Diagnostics Card

- [ ] TDR test initiates successfully
- [ ] Results display correctly
- [ ] Cable length accurate
- [ ] Fault detection works
- [ ] Pair-level details shown

### Link Monitoring

- [ ] Speed reported correctly
- [ ] Duplex reported correctly
- [ ] Carrier detect reliable
- [ ] State changes detected

### Network Discovery

- [ ] Interface detected correctly
- [ ] Statistics accurate
- [ ] Gateway ping works
- [ ] ARP table populated

## Issues Encountered

Document any problems:

**Issue 1**: [Description]

- Severity: Critical / Major / Minor
- Workaround: [If any]
- Error messages: [Paste relevant logs]
- Driver/Kernel interaction: [Notes]

**Issue 2**: [Description]

- ...

## Recommendation

**Overall Assessment**: Excellent / Good / Fair / Poor / Not Recommended

**TDR Cable Testing**: ✅ Fully Supported / ⚠️ Partial / ❌ Not Supported

- This is the **most critical feature** for cable diagnostics

**Use Case Recommendations**:

- ✅ **Cable Diagnostics**: [Yes/No - TDR support required]
- ✅ **Network Monitoring**: [Yes/No - reason]
- ✅ **Server/Enterprise**: [Yes/No - based on stability]
- ✅ **Development/Testing**: [Yes/No - reason]

**Summary**: [1-2 sentence recommendation, highlighting TDR support]

**Alternative Hardware** (if TDR not supported):

- Intel I350 (Quad-port GbE, excellent TDR)
- Intel I210 (Single-port GbE, good TDR)
- [Other recommendations]

## Additional Notes

[Any other observations, tips, or recommendations for future users]

---

**Tested By**: [Your GitHub username] **Test Date**: [YYYY-MM-DD] **Script Version**: [Git commit
hash or script version] **Cable Used**: [Cat5e/Cat6/Cat6a, length]
