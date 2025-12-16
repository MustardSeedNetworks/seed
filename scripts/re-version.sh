#!/bin/bash
# re-version.sh - Re-tag repository with correct semantic versioning
# Created for Issue #495
#
# DANGER: This script deletes all tags and re-creates them!
# Only run after creating backups per Phase 1 in issue #495

set -e  # Exit on error

echo "=== The Seed Re-Versioning Script ==="
echo ""
echo "This script will:"
echo "  1. Delete all local tags"
echo "  2. Re-create tags in chronological order"
echo "  3. (You must manually push to remote)"
echo ""
read -p "Have you completed Phase 1 backup? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
  echo "Aborting. Please complete Phase 1 backup first."
  exit 1
fi

echo ""
echo "Step 1: Deleting all local tags..."
git tag -l | xargs git tag -d

echo ""
echo "Step 2: Creating new tags in chronological order..."

# Array of commits and their new semantic versions
declare -A VERSION_MAP=(
  # Initial releases (v0.0.0 - v0.11.x) keep as-is
  ["f1762a7"]="v0.0.0"   # Initial setup
  ["ec3e019"]="v0.1.0"   # Core infrastructure
  ["0662868"]="v0.2.0"   # Switch card + discovery
  ["ae9010d"]="v0.3.0"   # DNS + DHCP
  ["39f122a"]="v0.4.0"   # Gateway ping
  ["81faa45"]="v0.4.1"   # VLAN detection
  ["4263df7"]="v0.5.0"   # WiFi + Cable TDR
  ["aff5769"]="v0.6.0"   # Settings drawer
  ["44b80e3"]="v0.6.1"   # JSON export
  ["380776a"]="v0.6.2"   # Theme improvements
  ["77970ef"]="v0.7.0"   # Speedtest card
  ["150682e"]="v0.7.1"   # Health checks
  ["8f9976d"]="v0.7.2"   # Settings persistence fix
  ["3579b9d"]="v0.7.3"   # Health checks fix
  ["487ff45"]="v0.7.4"   # HTTP URL prefix fix
  ["f075bc1"]="v0.7.5"   # HTTPS fallback
  ["d3685ff"]="v0.7.6"   # Health status fix
  ["bfb3571"]="v0.7.7"   # DHCP timing fix
  ["16882f7"]="v0.7.8"   # HTTP display improvements
  ["af9eb52"]="v0.8.0"   # iperf3 LAN testing
  ["5321189"]="v0.8.1"   # iperf3 bundling
  ["b9dbe97"]="v0.8.2"   # iperf3 settings
  ["f9c4bae"]="v0.8.3"   # FAB for all tests
  ["8c4f1e8"]="v0.8.4"   # Test suite + CI
  ["a49ccae"]="v0.8.5"   # FAB options + DNS
  ["5b975e9"]="v0.8.6"   # IPv6 gateway
  ["bb15de4"]="v0.8.7"   # Auto-save settings
  ["afb47c9"]="v0.8.8"   # FAB spinner fix
  ["7ced6d7"]="v0.8.9"   # DNS fix
  ["e949429"]="v0.9.0"   # Security hardening
  ["a5c609d"]="v0.9.1"   # Input validation
  ["af0a355"]="v0.9.2"   # Network discovery
  ["63264c4"]="v0.9.3"   # Discovery settings UI
  ["f30a6b4"]="v0.9.4"   # CORS fix
  ["60a6d0e"]="v0.9.5"   # Discovery API fix
  ["7082d15"]="v0.10.0"  # Discovery enhancements
  ["c489602"]="v0.10.1"  # Settings UI
  ["73fe7e0"]="v0.10.2"  # Native ICMP ping
  ["7f5d72a"]="v0.10.3"  # PING_ONLY fix
  ["d91e2b2"]="v0.10.4"  # Discovery method fix
  ["5fdb968"]="v0.10.5"  # Link detection fix
  ["8ca1936"]="v0.10.6"  # CORS config
  ["5fdb7de"]="v0.11.0"  # WebSocket updates
  ["d32da85"]="v0.11.1"  # Public IP card
  ["62f3d5e"]="v0.11.2"  # Public IP toggle
  ["2583c0d"]="v0.11.3"  # Auto-scan move
  ["e7524df"]="v0.11.4"  # go vet fixes
  ["2b526c5"]="v0.11.5"  # DNS per-server
  ["30dacd7"]="v0.11.6"  # CI simplify
  ["cd9aba3"]="v0.11.7"  # Security scanning
  ["e06b08d"]="v0.11.8"  # WebSocket safety
  ["54f9a2a"]="v0.12.0"  # Release please merge
  ["341cc3a"]="v0.12.1"  # Release please merge
  ["62ea766"]="v0.12.2"  # arm64 CI
  ["dd048d4"]="v0.12.3"  # Release build workflow
  ["3387321"]="v0.13.0"  # Frontend embed refactor
  ["8d3b2e1"]="v0.14.0"  # SettingsContext
  ["08f697e"]="v0.14.1"  # Card migration
  ["6d2a97d"]="v0.14.2"  # Threshold migration
  ["ec770e2"]="v0.14.3"  # Settings split begin
  ["34a8708"]="v0.14.4"  # WiFi/DNS sections
  ["a87eb8e"]="v0.14.5"  # Performance sections
  ["e72f64f"]="v0.15.0"  # BaseCard component
  ["1f7fc7c"]="v0.15.1"  # Remove custom events
  ["a19dce1"]="v0.15.2"  # Passive cards migration
  ["d5167d3"]="v0.16.0"  # API validation
  ["2405794"]="v0.17.0"  # -trimpath build
  ["17083d6"]="v0.17.1"  # Link flap tracking
  ["c301823"]="v0.18.0"  # TCP port prober
  ["0aaa0e5"]="v0.19.0"  # Traceroute
  ["7aabd73"]="v0.20.0"  # Port scanner
  ["10e4409"]="v0.20.1"  # Card rename/merge
  ["7cc53f2"]="v0.20.2"  # VLAN + MTU UI
  ["81c61ef"]="v0.20.3"  # Settings separation
  ["b3ab9a0"]="v0.20.4"  # SettingsDrawer wiring
  ["844100f"]="v0.20.5"  # Type standardization
  ["327096e"]="v0.20.6"  # Persist all settings
  ["4edd469"]="v0.20.7"  # Gateway route fix
  ["8d29d93"]="v0.21.0"  # ACME/Let's Encrypt
  ["4fd531c"]="v0.22.0"  # OS/service fingerprinting

  # THE FIX: These were v0.12.5-v0.12.11, now become v0.22.x-v0.23.x
  ["f9c928c"]="v0.22.1"  # Threshold tooltips (was v0.12.5)
  ["9396860"]="v0.22.2"  # Discovery summary (was v0.12.6)
  ["d91e1ea"]="v0.22.3"  # Speedtest gauge (was v0.12.7)
  ["b727d95"]="v0.22.4"  # System Health card (was v0.12.8)
  ["b1bf5fd"]="v0.22.5"  # iperf3 timeouts (was v0.12.9)
  ["35a6a6d"]="v0.22.6"  # Rogue DHCP (was v0.12.10)
  ["d022804"]="v0.22.7"  # CVE scanning (was v0.12.11)
  ["c7d581b"]="v0.23.0"  # IPv6 NDP (CURRENT)
)

# Apply all tags
tag_count=0
for commit in "${!VERSION_MAP[@]}"; do
  version="${VERSION_MAP[$commit]}"

  # Check if commit exists
  if ! git cat-file -e "$commit^{commit}" 2>/dev/null; then
    echo "WARNING: Commit $commit not found, skipping $version"
    continue
  fi

  echo "  Tagging $commit as $version"
  git tag -a "$version" "$commit" -m "Re-versioned: $version"
  ((tag_count++))
done

echo ""
echo "Step 3: Verification..."
echo "Created $tag_count tags"

echo ""
echo "Checking chronological order (last 20 tags):"
git log --oneline --decorate --all --reverse | grep "tag:" | tail -20

echo ""
echo "Current HEAD tag:"
git describe --tags

echo ""
echo "=== SUCCESS ==="
echo ""
echo "Next steps:"
echo "  1. Review the tag list above"
echo "  2. Delete remote tags: See Phase 2 in issue #495"
echo "  3. Push new tags: git push origin --tags"
echo "  4. Verify: git ls-remote --tags origin | sort -t '/' -k 3 -V"
echo ""
