/**
 * WiFiSurveyHelp.tsx
 *
 * Purpose: Comprehensive help documentation for WiFi Site Survey feature.
 * Provides user guidance on survey creation, floor plan uploads, heatmap visualization,
 * and interpreting survey results.
 *
 * Key Features:
 * - Overview section: WiFi survey purpose and use cases
 * - Requirements section: Prerequisites for running surveys
 * - Floor plan guide: How to upload and configure floor plans
 * - Sample collection: Passive, active, and throughput measurement modes
 * - Heatmap help: Interpreting color-coded signal strength and performance maps
 * - Troubleshooting: Common issues and resolution steps
 * - Data export: How to export survey results
 *
 * Usage:
 * ```typescript
 * <HelpModal sections={[...WIFI_SURVEY_HELP]} />
 * ```
 *
 * Dependencies: None (pure data constants)
 * Structure: Array of HelpSection with items containing question/answer pairs
 */

interface HelpItem {
  question: string;
  answer: string;
}

interface HelpSection {
  title: string;
  items: HelpItem[];
}

export const WIFI_SURVEY_HELP: HelpSection[] = [
  {
    title: 'Overview',
    items: [
      {
        question: 'What is WiFi Site Survey?',
        answer:
          'WiFi Site Survey allows you to create detailed coverage maps of your wireless network by measuring signal strength, performance, and connectivity at different physical locations. You can upload a floor plan and click on locations to record measurements, creating heatmaps that visualize WiFi coverage, signal strength, throughput, and problem areas.',
      },
      {
        question: 'When should I use WiFi Survey?',
        answer:
          'Use WiFi Survey to:\n• Optimize access point placement for maximum coverage\n• Identify dead zones and weak signal areas\n• Troubleshoot connectivity issues in specific locations\n• Plan network expansions or upgrades\n• Document network performance for compliance or SLA requirements\n• Validate roaming behavior between access points',
      },
      {
        question: 'Requirements',
        answer:
          'To use WiFi Site Survey:\n• Your active network interface must be WiFi (not Ethernet)\n• The WiFi Survey card will only appear when connected via WiFi\n• For Throughput surveys, you need an iperf3 server on your network\n• Recommended: A floor plan image (PNG, JPG, JPEG, GIF, WEBP, or SVG)',
      },
    ],
  },
  {
    title: 'Survey Modes',
    items: [
      {
        question: 'Passive Scan Mode',
        answer:
          'Passive mode scans for all visible WiFi networks at each measurement point without connecting to them.\n\nWhat it measures:\n• All visible SSIDs and BSSIDs (access points)\n• Signal strength (RSSI) for each network\n• Channel and frequency information\n• Network security type\n\nBest for:\n• Initial site surveys to see all available networks\n• Identifying channel interference from neighboring networks\n• Detecting rogue access points\n• Planning which channels to use for your network\n\nNote: Passive scans do NOT test your actual connection quality - they just show what networks are visible.',
      },
      {
        question: 'Active Monitoring Mode',
        answer:
          'Active mode monitors your current WiFi connection at each measurement point.\n\nWhat it measures:\n• Current SSID and BSSID (connected access point)\n• Real-time signal strength (RSSI)\n• Current data rate (Mbps)\n• Roaming events (when you switch between access points)\n\nBest for:\n• Testing roaming behavior as you move around\n• Identifying where signal drops below acceptable levels\n• Finding optimal access point placement\n• Troubleshooting specific coverage issues on your network\n\nNote: You must stay connected to your WiFi network throughout the survey.',
      },
      {
        question: 'Throughput Testing Mode',
        answer:
          'Throughput mode performs actual speed tests at each measurement point using iperf3.\n\nWhat it measures:\n• Download speed (Mbps)\n• Upload speed (Mbps)\n• Latency (milliseconds)\n• Jitter (milliseconds)\n• Packet loss (percentage)\n• Signal strength (RSSI)\n• Connected BSSID\n\nBest for:\n• Testing real-world application performance\n• Identifying bandwidth bottlenecks\n• Validating network design meets performance requirements\n• Troubleshooting slow performance in specific areas\n\nRequirements:\n• An iperf3 server must be running on your network\n• Configure the server IP:port in survey settings\n• Each test takes ~3-5 seconds to complete\n\nNote: Throughput testing is the most comprehensive but slowest survey method.',
      },
    ],
  },
  {
    title: 'Creating a Survey',
    items: [
      {
        question: 'How do I create a new survey?',
        answer:
          "1. Ensure you're connected via WiFi (the WiFi Survey card only appears on WiFi interfaces)\n2. Click the '+ New' button in the WiFi Survey card\n3. Enter a descriptive survey name (e.g., 'Office Floor 2', 'Warehouse Coverage')\n4. Select a survey mode:\n   • Passive Scan - See all visible networks\n   • Active Monitoring - Test your current connection\n   • Throughput Testing - Measure actual speeds (requires iperf3 server)\n5. Click 'Create Survey'\n\nThe survey will be created in 'Created' status, ready for you to add a floor plan and start measurements.",
      },
      {
        question: 'How do I upload a floor plan?',
        answer:
          "After creating a survey:\n1. Click on the survey to open it\n2. Click 'Upload Floor Plan' or drag-and-drop an image\n3. Supported image formats:\n   • PNG (recommended for floor plans)\n   • JPG/JPEG\n   • GIF\n   • WEBP\n   • SVG\n\n4. The image will be displayed, and you can click on locations to record measurements\n\nTips:\n• Use a high-resolution image for better accuracy\n• Ensure the floor plan shows walls, doors, and major obstacles\n• Remove any unnecessary details to keep the file size manageable\n• PNG format is recommended for clarity and transparency support",
      },
      {
        question: 'Can I create a survey without a floor plan?',
        answer:
          "Yes! You can create a survey without a floor plan and still record measurements. However, without a floor plan:\n• You won't see a visual heatmap overlay\n• Measurements will be listed in a table format\n• It's harder to correlate measurements with physical locations\n\nFor best results, we recommend uploading a floor plan. Even a simple sketch or photo of your space works better than no visual reference.",
      },
    ],
  },
  {
    title: 'Conducting a Survey',
    items: [
      {
        question: 'How do I take measurements?',
        answer:
          "1. Start the survey by clicking the Play (▶) button\n2. The survey status changes to 'In Progress'\n3. Click on your current physical location on the floor plan\n4. Wait for the measurement to complete (~1-5 seconds depending on mode)\n5. Move to the next location and repeat\n6. Click Pause (⏸) to temporarily stop, or Complete (✓) when finished\n\nBest practices:\n• Take measurements in a grid pattern for even coverage\n• Spend ~30 seconds at each location before measuring (allows signal to stabilize)\n• Take extra measurements in problem areas or high-traffic zones\n• For roaming tests, walk slowly between points while monitoring signal\n• Avoid taking measurements while downloading or streaming (can affect results)",
      },
      {
        question: 'How many sample points should I take?',
        answer:
          "The number of samples depends on:\n\n• Space size: Larger areas need more points\n• Coverage goals: Critical areas need denser sampling\n• Survey mode: Throughput tests require fewer points due to time\n\nGeneral guidelines:\n• Small office (500-1000 sq ft): 15-30 points\n• Medium space (1000-3000 sq ft): 30-60 points\n• Large space (3000+ sq ft): 60-100+ points\n• Grid spacing: 10-20 feet apart for active/passive, 15-30 feet for throughput\n\nRemember: Quality over quantity. It's better to have 30 well-placed measurements than 100 random ones.",
      },
      {
        question: 'Can I pause and resume a survey?',
        answer:
          "Yes! You can pause a survey at any time:\n\n1. Click the Pause (⏸) button while survey is in progress\n2. Status changes to 'Paused'\n3. Your measurements are saved\n4. Click Play (▶) to resume from where you left off\n5. Click Complete (✓) if you're finished and don't want to add more points\n\nThis is useful when:\n• You need to take a break during a large survey\n• Network conditions change (wait for interference to clear)\n• You want to review preliminary results before continuing\n• Equipment needs recharging or repositioning",
      },
    ],
  },
  {
    title: 'Viewing Results',
    items: [
      {
        question: 'How do I view survey results?',
        answer:
          '1. Click on any survey in the WiFi Survey card\n2. The survey details dialog opens showing:\n   • Survey metadata (type, status, sample count, dates)\n   • Floor plan with measurement points overlaid\n   • Heatmap visualization (if available)\n   • Sample point details table\n\n3. For completed surveys, you can:\n   • Switch between different metrics (RSSI, throughput, etc.)\n   • View individual sample point data\n   • Export results (coming soon)\n   • Compare before/after changes',
      },
      {
        question: 'What do the heatmap colors mean?',
        answer:
          'Heatmaps use color gradients to show metric values:\n\nSignal Strength (RSSI):\n• Green (-30 to -50 dBm): Excellent signal\n• Yellow (-50 to -70 dBm): Good signal\n• Orange (-70 to -80 dBm): Fair signal\n• Red (-80+ dBm): Poor signal / dead zone\n\nThroughput (Mbps):\n• Green (80-100%+ of expected): Excellent performance\n• Yellow (50-80%): Good performance\n• Orange (25-50%): Fair performance\n• Red (<25%): Poor performance\n\nThe heatmap interpolates between measurement points to estimate coverage in unmeasured areas.',
      },
      {
        question: 'Can I export survey data?',
        answer:
          'Survey export functionality is coming soon. Future versions will support:\n• CSV export of all sample points\n• PDF reports with heatmap images\n• Comparison reports (before/after changes)\n• Integration with professional tools\n\nCurrently, you can:\n• View all measurements in the web interface\n• Take screenshots of heatmaps\n• Access raw data via the API (/api/canopy/survey endpoints)',
      },
    ],
  },
  {
    title: 'Troubleshooting',
    items: [
      {
        question: "Why don't I see the WiFi Survey card?",
        answer:
          "The WiFi Survey card only appears when your active network interface is WiFi. If you don't see it:\n\n1. Check your current interface (shown in the top-right header)\n2. If on Ethernet, switch to WiFi:\n   • Click Settings (gear icon)\n   • Go to Interface settings\n   • Select your WiFi interface\n   • Click Apply\n\n3. Connect to WiFi and refresh the page\n\nNote: You cannot conduct WiFi surveys while connected via Ethernet, even if you have a WiFi adapter.",
      },
      {
        question: "Survey creation fails with 'requires iperf3'",
        answer:
          "This error occurs when creating a Throughput survey without an iperf3 server configured.\n\nTo fix:\n1. Set up an iperf3 server on your network:\n   • On Linux/Mac: `iperf3 -s`\n   • On Windows: Download from iperf.fr and run `iperf3.exe -s`\n\n2. Configure the server in The Seed:\n   • Settings > Performance Testing > iperf3 Server\n   • Enter server IP:port (default port is 5201)\n   • Example: 192.168.1.100:5201\n\nAlternatively, use Passive or Active survey modes which don't require iperf3.",
      },
      {
        question: 'Measurements are inconsistent or varying',
        answer:
          'WiFi signal naturally varies due to interference, movement, and environmental factors. To get consistent results:\n\n• Wait 30-60 seconds at each location before measuring (signal needs time to stabilize)\n• Avoid measuring during heavy network usage\n• Turn off or move away from sources of interference (microwaves, Bluetooth devices)\n• Keep your device in the same orientation at each point\n• Take multiple measurements at critical locations and average them\n• Conduct surveys at the same time of day for before/after comparisons\n• Ensure no one is moving large metal objects during the survey',
      },
      {
        question: "Floor plan image won't upload",
        answer:
          'If floor plan upload fails:\n\n1. Check file format - supported types:\n   • PNG, JPG, JPEG, GIF, WEBP, SVG\n\n2. Check file size:\n   • Maximum recommended: 10MB\n   • For faster loading, resize to 2000x2000 pixels or less\n\n3. Check image corruption:\n   • Try opening the image in another program\n   • Re-save or export the image\n   • Convert to PNG if using an unusual format\n\n4. Browser issues:\n   • Try a different browser\n   • Clear browser cache\n   • Disable browser extensions that might block uploads',
      },
    ],
  },
  {
    title: 'Best Practices',
    items: [
      {
        question: 'Survey workflow recommendations',
        answer:
          'For best results, follow this workflow:\n\n1. Planning (before survey):\n   • Obtain or create a floor plan\n   • Decide which mode matches your goals\n   • Note existing problem areas\n   • Schedule during low-usage times\n\n2. Initial baseline (Passive mode):\n   • Scan for all visible networks\n   • Identify interference sources\n   • Check channel overlap\n\n3. Coverage validation (Active mode):\n   • Test your actual network\n   • Walk a roaming path\n   • Measure at workstations/high-use areas\n\n4. Performance verification (Throughput mode):\n   • Test bandwidth at key locations\n   • Focus on high-demand areas\n   • Validate against requirements\n\n5. Documentation & action:\n   • Save completed surveys\n   • Identify issues and fixes\n   • Re-survey after changes to validate improvements',
      },
    ],
  },
];
