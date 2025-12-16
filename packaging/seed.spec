Name:       seed
Version:    __VERSION__
Release:    1%{?dist}
Summary:    Portable Network Diagnostic Tool with Real-Time Web UI
License:    BSL 1.1
URL:        https://github.com/krisarmstrong/seed
BuildArch:  __RPM_ARCH__

Requires:   libpcap, systemd

%description
The Seed is a professional-grade network diagnostic appliance designed for network technicians and engineers.
Plug it into any network jack and instantly see link status, switch information, DHCP details, DNS health, and gateway connectivity through a modern web interface.

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/local/bin
mkdir -p %{buildroot}/etc/seed
mkdir -p %{buildroot}/lib/systemd/system
mkdir -p %{buildroot}/usr/share/seed

# Copy binaries
install -m 755 %{_repo_root}/dist/seed-linux-__ARCHITECTURE__ %{buildroot}/usr/local/bin/seed
install -m 755 %{_repo_root}/bin/iperf3-linux-__ARCHITECTURE__ %{buildroot}/usr/local/bin/iperf3

# Copy config files
cp -r %{_repo_root}/configs/* %{buildroot}/etc/seed/

# Copy web assets
cp -r %{_repo_root}/web/dist %{buildroot}/usr/share/seed/web

# Copy systemd service file
install -m 644 %{_repo_root}/packaging/seed.service %{buildroot}/lib/systemd/system/seed.service

%files
/usr/local/bin/seed
/usr/local/bin/iperf3
/etc/seed
/lib/systemd/system/seed.service
/usr/share/seed/web

%post
%systemd_post seed.service

%preun
%systemd_preun seed.service

%postun
%systemd_postun_with_restart seed.service
