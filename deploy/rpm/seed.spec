Name:       seed
Version:    __VERSION__
Release:    1%{?dist}
Summary:    The Seed - Network Diagnostic Tool by Mustard Seed Networks
License:    BSL 1.1
URL:        https://github.com/krisarmstrong/seed
BuildArch:  __ARCHITECTURE__

Requires:   libpcap, systemd, libcap

%description
The Seed is a professional-grade network diagnostic appliance designed
for network technicians and engineers. Plug it into any network jack and
instantly see link status, switch information, DHCP details, DNS health,
and gateway connectivity through a modern web interface.

Features:
- Real-time network diagnostics via web UI
- WiFi survey and heatmap generation
- Speed testing with iPerf3 integration
- SNMP device discovery
- DHCP rogue detection
- Cable diagnostics (TDR)
- Vulnerability scanning (CVE/CISA KEV)

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/lib/systemd/system
mkdir -p %{buildroot}/etc/seed
mkdir -p %{buildroot}/var/lib/seed
mkdir -p %{buildroot}/var/log/seed

# Copy binary (single binary with embedded assets)
install -m 755 %{_repo_root}/seed %{buildroot}/usr/bin/seed

# Copy systemd service file
install -m 644 %{_repo_root}/deploy/deb/seed.service %{buildroot}/usr/lib/systemd/system/seed.service

%files
%attr(755, root, root) /usr/bin/seed
%attr(644, root, root) /usr/lib/systemd/system/seed.service
%dir %attr(750, seed, seed) /etc/seed
%dir %attr(750, seed, seed) /var/lib/seed
%dir %attr(750, seed, seed) /var/log/seed

%pre
# Create service user and group
getent group seed >/dev/null || groupadd -r seed
getent passwd seed >/dev/null || \
    useradd -r -g seed -d /var/lib/seed -s /sbin/nologin \
    -c "The Seed Network Diagnostic Tool" seed
exit 0

%post
# Set ownership of directories
chown -R seed:seed /etc/seed /var/lib/seed /var/log/seed

# Set capabilities for raw socket access
# - CAP_NET_RAW: Required for ICMP ping, ARP scanning, packet capture
# - CAP_NET_ADMIN: Required for ethtool link configuration, interface control
/usr/sbin/setcap 'cap_net_raw,cap_net_admin=+ep' /usr/bin/seed || true

%systemd_post seed.service

%preun
%systemd_preun seed.service

%postun
%systemd_postun_with_restart seed.service

# On complete removal (not upgrade), optionally remove user/group
if [ $1 -eq 0 ]; then
    # Full removal
    userdel seed 2>/dev/null || true
    groupdel seed 2>/dev/null || true
fi

%changelog
* Fri Dec 27 2024 Kris Armstrong <kris@mustardseednetworks.com>
- Streamlined packaging with FHS-compliant paths
- Single binary with embedded frontend assets
- Added default profile seeding on fresh database
- Updated systemd service with security hardening
