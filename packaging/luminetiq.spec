Name:       luminetiq
Version:    __VERSION__
Release:    1%{?dist}
Summary:    Portable Network Diagnostic Tool with Real-Time Web UI
License:    BSL 1.1
URL:        https://github.com/krisarmstrong/netscope
BuildArch:  __RPM_ARCH__

Requires:   libpcap, systemd

%description
LuminetIQ is a professional-grade network diagnostic appliance designed for network technicians and engineers.
Plug it into any network jack and instantly see link status, switch information, DHCP details, DNS health, and gateway connectivity through a modern web interface.

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/local/bin
mkdir -p %{buildroot}/etc/luminetiq
mkdir -p %{buildroot}/lib/systemd/system
mkdir -p %{buildroot}/usr/share/luminetiq

# Copy binaries
install -m 755 %{_repo_root}/dist/luminetiq-linux-__ARCHITECTURE__ %{buildroot}/usr/local/bin/luminetiq
install -m 755 %{_repo_root}/bin/iperf3-linux-__ARCHITECTURE__ %{buildroot}/usr/local/bin/iperf3

# Copy config files
cp -r %{_repo_root}/configs/* %{buildroot}/etc/luminetiq/

# Copy web assets
cp -r %{_repo_root}/web/dist %{buildroot}/usr/share/luminetiq/web

# Copy systemd service file
install -m 644 %{_repo_root}/packaging/luminetiq.service %{buildroot}/lib/systemd/system/luminetiq.service

%files
/usr/local/bin/luminetiq
/usr/local/bin/iperf3
/etc/luminetiq
/lib/systemd/system/luminetiq.service
/usr/share/luminetiq/web

%post
%systemd_post luminetiq.service

%preun
%systemd_preun luminetiq.service

%postun
%systemd_postun_with_restart luminetiq.service
