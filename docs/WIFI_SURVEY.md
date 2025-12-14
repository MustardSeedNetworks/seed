# WiFi Site Survey

## Overview

WiFi Site Survey is a powerful feature in LuminetIQ that allows you to create detailed coverage maps of your wireless network by measuring signal strength, performance, and connectivity at different physical locations. You can upload a floor plan and click on locations to record measurements, creating heatmaps that visualize WiFi coverage, signal strength, throughput, and problem areas.

## Prerequisites

### Required

- **WiFi Interface**: Your active network interface must be WiFi (not Ethernet)
- **WiFi Connection**: You must be connected to a WiFi network during the survey

The WiFi Survey card will only appear in the LuminetIQ dashboard when you are connected via WiFi.

### Optional

- **Floor Plan Image**: PNG, JPG, JPEG, GIF, WEBP, or SVG file showing your space layout
  - Recommended: High-resolution PNG (2000x2000 pixels or less for optimal loading)
  - Should clearly show walls, doors, and major obstacles
- **iperf3 Server**: Required only for Throughput survey mode
  - Must be running on your network
  - Default port: 5201

## When to Use WiFi Survey

WiFi Site Survey is ideal for:

- **Optimizing Access Point Placement**: Determine the best locations for maximum coverage
- **Identifying Dead Zones**: Find areas with weak or no signal
- **Troubleshooting Connectivity**: Diagnose issues in specific locations
- **Network Planning**: Plan expansions or upgrades with data-driven insights
- **Compliance & SLA**: Document network performance for requirements
- **Roaming Validation**: Test how devices hand off between access points

## Survey Modes

### Passive Scan Mode

**What it does**: Scans for all visible WiFi networks at each measurement point without connecting to them.

**What it measures**:
- All visible SSIDs and BSSIDs (access points)
- Signal strength (RSSI) for each network
- Channel and frequency information
- Network security type

**Best for**:
- Initial site surveys to see all available networks
- Identifying channel interference from neighboring networks
- Detecting rogue access points
- Planning which channels to use for your network

**Note**: Passive scans do NOT test your actual connection quality - they just show what networks are visible.

### Active Monitoring Mode

**What it does**: Monitors your current WiFi connection at each measurement point.

**What it measures**:
- Current SSID and BSSID (connected access point)
- Real-time signal strength (RSSI)
- Current data rate (Mbps)
- Roaming events (when you switch between access points)

**Best for**:
- Testing roaming behavior as you move around
- Identifying where signal drops below acceptable levels
- Finding optimal access point placement
- Troubleshooting specific coverage issues on your network

**Note**: You must stay connected to your WiFi network throughout the survey.

### Throughput Testing Mode

**What it does**: Performs actual speed tests at each measurement point using iperf3.

**What it measures**:
- Download speed (Mbps)
- Upload speed (Mbps)
- Latency (milliseconds)
- Jitter (milliseconds)
- Packet loss (percentage)
- Signal strength (RSSI)
- Connected BSSID

**Best for**:
- Testing real-world application performance
- Identifying bandwidth bottlenecks
- Validating network design meets performance requirements
- Troubleshooting slow performance in specific areas

**Requirements**:
- An iperf3 server must be running on your network
- Configure the server IP:port in survey settings (Settings → Performance Testing → iperf3 Server)
- Each test takes ~3-5 seconds to complete

**Note**: Throughput testing is the most comprehensive but slowest survey method.

## Creating a Survey

### Step 1: Ensure WiFi Connection

1. Verify you're connected via WiFi (not Ethernet)
2. The WiFi Survey card should appear in the dashboard
3. Check your current interface in the top-right header

If on Ethernet, switch to WiFi:
1. Click Settings (gear icon)
2. Go to Interface settings
3. Select your WiFi interface
4. Click Apply
5. Connect to WiFi and refresh the page

### Step 2: Create New Survey

1. Click the **+ New** button in the WiFi Survey card
2. Enter a descriptive survey name
   - Examples: "Office Floor 2", "Warehouse Coverage", "Main Building Baseline"
3. Select a survey mode:
   - **Passive Scan** - See all visible networks
   - **Active Monitoring** - Test your current connection
   - **Throughput Testing** - Measure actual speeds (requires iperf3 server)
4. Click **Create Survey**

The survey will be created in "Created" status, ready for you to add a floor plan and start measurements.

### Step 3: Upload Floor Plan (Optional)

While you can create a survey without a floor plan, having one provides much better visualization.

1. Click on the survey to open it
2. Click **Upload Floor Plan** or drag-and-drop an image
3. Supported formats:
   - PNG (recommended for floor plans)
   - JPG/JPEG
   - GIF
   - WEBP
   - SVG

**Tips for floor plans**:
- Use a high-resolution image for better accuracy
- Ensure the floor plan shows walls, doors, and major obstacles
- Remove unnecessary details to keep file size manageable
- PNG format recommended for clarity and transparency support

### Step 4: Start Survey

1. Click the **Play (▶)** button to start
2. Survey status changes to "In Progress"
3. You're now ready to take measurements

## Conducting a Survey

### Taking Measurements

1. Walk to a physical location in your space
2. Wait 30-60 seconds for the signal to stabilize
3. Click on that location on the floor plan (or use the interface if no floor plan)
4. Wait for the measurement to complete (~1-5 seconds depending on mode)
5. The measurement point appears on the map
6. Move to the next location and repeat

**Pro tips**:
- Take measurements in a grid pattern for even coverage
- Spend ~30 seconds at each location before measuring (allows signal to stabilize)
- Take extra measurements in problem areas or high-traffic zones
- For roaming tests, walk slowly between points while monitoring signal
- Avoid measuring during heavy network usage or downloads

### How Many Measurement Points?

The number of samples depends on:
- **Space size**: Larger areas need more points
- **Coverage goals**: Critical areas need denser sampling
- **Survey mode**: Throughput tests require fewer points due to time

**General guidelines**:
- **Small office (500-1000 sq ft)**: 15-30 points
- **Medium space (1000-3000 sq ft)**: 30-60 points
- **Large space (3000+ sq ft)**: 60-100+ points
- **Grid spacing**:
  - Active/Passive: 10-20 feet apart
  - Throughput: 15-30 feet apart

**Remember**: Quality over quantity. It's better to have 30 well-placed measurements than 100 random ones.

### Pausing and Resuming

You can pause a survey at any time:

1. Click the **Pause (⏸)** button while survey is in progress
2. Status changes to "Paused"
3. Your measurements are saved
4. Click **Play (▶)** to resume from where you left off
5. Click **Complete (✓)** if you're finished and don't want to add more points

**Useful when**:
- You need to take a break during a large survey
- Network conditions change (wait for interference to clear)
- You want to review preliminary results before continuing
- Equipment needs recharging or repositioning

## Viewing Results

### Opening Survey Details

1. Click on any survey in the WiFi Survey card
2. The survey details dialog opens showing:
   - Survey metadata (type, status, sample count, dates)
   - Floor plan with measurement points overlaid
   - Heatmap visualization (if available)
   - Sample point details table

### Understanding Heatmaps

Heatmaps use color gradients to show metric values across your space:

**Signal Strength (RSSI)**:
- **Green (-30 to -50 dBm)**: Excellent signal
- **Yellow (-50 to -70 dBm)**: Good signal
- **Orange (-70 to -80 dBm)**: Fair signal
- **Red (-80+ dBm)**: Poor signal / dead zone

**Throughput (Mbps)**:
- **Green (80-100%+ of expected)**: Excellent performance
- **Yellow (50-80%)**: Good performance
- **Orange (25-50%)**: Fair performance
- **Red (<25%)**: Poor performance

The heatmap interpolates between measurement points to estimate coverage in unmeasured areas.

### Exporting Data

Survey export functionality is coming soon. Future versions will support:
- CSV export of all sample points
- PDF reports with heatmap images
- Comparison reports (before/after changes)
- Integration with professional tools

Currently, you can:
- View all measurements in the web interface
- Take screenshots of heatmaps
- Access raw data via the API (`/api/survey` endpoints)

## Troubleshooting

### WiFi Survey Card Not Visible

**Problem**: The WiFi Survey card doesn't appear in the dashboard.

**Solution**:
1. Check your current interface (shown in the top-right header)
2. If on Ethernet, switch to WiFi:
   - Click Settings (gear icon)
   - Go to Interface settings
   - Select your WiFi interface
   - Click Apply
3. Connect to WiFi and refresh the page

**Note**: You cannot conduct WiFi surveys while connected via Ethernet, even if you have a WiFi adapter.

### Survey Creation Fails with "requires iperf3"

**Problem**: Creating a Throughput survey fails with error about iperf3.

**Solution**:
1. Set up an iperf3 server on your network:
   - **Linux/Mac**: `iperf3 -s`
   - **Windows**: Download from [iperf.fr](https://iperf.fr/) and run `iperf3.exe -s`
2. Configure the server in LuminetIQ:
   - Settings → Performance Testing → iperf3 Server
   - Enter server IP:port (default port is 5201)
   - Example: `192.168.1.100:5201`

**Alternative**: Use Passive or Active survey modes which don't require iperf3.

### Measurements Are Inconsistent

**Problem**: Signal strength or throughput varies significantly between measurements.

**Explanation**: WiFi signal naturally varies due to interference, movement, and environmental factors.

**Solutions**:
- Wait 30-60 seconds at each location before measuring (signal needs time to stabilize)
- Avoid measuring during heavy network usage
- Turn off or move away from sources of interference:
  - Microwaves
  - Bluetooth devices
  - Other 2.4 GHz devices
- Keep your device in the same orientation at each point
- Take multiple measurements at critical locations and average them
- Conduct surveys at the same time of day for before/after comparisons
- Ensure no one is moving large metal objects during the survey

### Floor Plan Won't Upload

**Problem**: Floor plan image upload fails.

**Solutions**:

1. **Check file format** - Supported types:
   - PNG, JPG, JPEG, GIF, WEBP, SVG

2. **Check file size**:
   - Maximum recommended: 10MB
   - For faster loading, resize to 2000x2000 pixels or less

3. **Check image corruption**:
   - Try opening the image in another program
   - Re-save or export the image
   - Convert to PNG if using an unusual format

4. **Browser issues**:
   - Try a different browser
   - Clear browser cache
   - Disable browser extensions that might block uploads

## Best Practices

### Recommended Survey Workflow

For best results, follow this workflow:

#### 1. Planning (Before Survey)
- Obtain or create a floor plan
- Decide which mode matches your goals
- Note existing problem areas
- Schedule during low-usage times

#### 2. Initial Baseline (Passive Mode)
- Scan for all visible networks
- Identify interference sources
- Check channel overlap
- Document current state

#### 3. Coverage Validation (Active Mode)
- Test your actual network
- Walk a roaming path
- Measure at workstations/high-use areas
- Verify coverage meets requirements

#### 4. Performance Verification (Throughput Mode)
- Test bandwidth at key locations
- Focus on high-demand areas
- Validate against requirements
- Document bottlenecks

#### 5. Documentation & Action
- Save completed surveys
- Identify issues and fixes
- Make network adjustments
- Re-survey after changes to validate improvements

### Survey Tips

**Before starting**:
- Charge your laptop/device fully
- Close bandwidth-intensive applications
- Ensure iperf3 server is running (for throughput surveys)
- Print or have floor plan available for reference

**During survey**:
- Be systematic - use a grid pattern
- Label problem areas as you find them
- Take photos of physical obstacles if relevant
- Note any interference sources you observe

**After completion**:
- Review heatmap for obvious gaps
- Add supplementary measurements in transition zones
- Compare results with known issues
- Document findings for stakeholders

## API Reference

For programmatic access to survey data, use these endpoints:

```
POST   /api/survey/create          - Create new survey
GET    /api/survey/list            - List all surveys
GET    /api/survey?id={id}         - Get survey details
DELETE /api/survey/delete?id={id}  - Delete survey
POST   /api/survey/start?id={id}   - Start survey
POST   /api/survey/pause?id={id}   - Pause survey
POST   /api/survey/complete?id={id}- Complete survey
POST   /api/survey/sample?id={id}  - Add sample point
POST   /api/survey/floorplan?id={id} - Upload floor plan
GET    /api/survey/heatmap?id={id}&metric=rssi - Get heatmap data
```

All endpoints require authentication via JWT token.

## Technical Details

### Survey Data Model

```typescript
interface Survey {
  id: string;
  name: string;
  surveyType: "passive" | "active" | "throughput";
  status: "created" | "inProgress" | "paused" | "completed";
  sampleCount: number;
  createdAt: string;
  updatedAt: string;
  floorPlan?: FloorPlan;
}

interface SamplePoint {
  x: number;         // pixel coordinates on floor plan
  y: number;
  timestamp: string;
  data: PassiveSample | ActiveSample | ThroughputSample;
}

interface PassiveSample {
  networks: ScannedNetwork[];  // All visible APs
}

interface ActiveSample {
  ssid: string;
  bssid: string;
  rssi: number;
  dataRate: number;
  roamingEvent: boolean;
}

interface ThroughputSample {
  ssid: string;
  bssid: string;
  rssi: number;
  downloadMbps: number;
  uploadMbps: number;
  latency: number;
  jitter: number;
  packetLoss: number;
}
```

### Heatmap Generation

Heatmaps use **Inverse Distance Weighting (IDW)** interpolation to estimate signal strength or throughput between measurement points. This provides a smooth gradient visualization of coverage across the entire floor plan.

## Support

For issues or questions:
- GitHub: [github.com/krisarmstrong/luminetiq/issues](https://github.com/krisarmstrong/luminetiq/issues)
- Check logs: `tail -f luminetiq.log`
