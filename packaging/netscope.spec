Name:       netscope
Version:    __VERSION__
Release:    1%{?dist}
Summary:    Portable Network Diagnostic Tool with Real-Time Web UI
License:    BSL 1.1
URL:        https://github.com/krisarmstrong/netscope
BuildArch:  __RPM_ARCH__

Requires:   libpcap, systemd

%description
NetScope is a professional-grade network diagnostic appliance designed for network technicians and engineers.
Plug it into any network jack and instantly see link status, switch information, DHCP details, DNS health, and gateway connectivity through a modern web interface.

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/local/bin
mkdir -p %{buildroot}/etc/netscope
mkdir -p %{buildroot}/lib/systemd/system
mkdir -p %{buildroot}/usr/share/netscope

# Copy binaries
install -m 755 %{_repo_root}/dist/netscope-linux-__ARCHITECTURE__ %{buildroot}/usr/local/bin/netscope
install -m 755 %{_repo_root}/bin/iperf3-linux-__ARCHITECTURE__ %{buildroot}/usr/local/bin/iperf3

# Copy config files
cp -r %{_repo_root}/configs/* %{buildroot}/etc/netscope/

# Copy web assets
cp -r %{_repo_root}/web/dist %{buildroot}/usr/share/netscope/web

# Copy systemd service file
install -m 644 %{_repo_root}/packaging/netscope.service %{buildroot}/lib/systemd/system/netscope.service

%files
/usr/local/bin/netscope
/usr/local/bin/iperf3
/etc/netscope
/lib/systemd/system/netscope.service
/usr/share/netscope/web

%post
%systemd_post netscope.service

%preun
%systemd_preun netscope.service

%postun
%systemd_postun_with_restart netscope.service
