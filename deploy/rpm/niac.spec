Name:       niac
Version:    __VERSION__
Release:    1%{?dist}
Summary:    NiAC - Network Infrastructure Agent Cluster
License:    Proprietary
URL:        https://github.com/krisarmstrong/niac
BuildArch:  __ARCHITECTURE__

Requires:   libpcap, systemd, libcap
Requires(pre): shadow-utils
Provides: user(niac)
Provides: group(niac)

%description
NiAC (Network In A Can) is a network device simulator supporting:
- SNMP agent simulation with walk file replay
- Device template management
- Multi-device orchestration
- WebUI for configuration and control
- Packet capture and traffic generation
- Protocol simulation (ARP, ICMP, DHCP, DNS, HTTP, FTP, LLDP, CDP)

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/lib/systemd/system
mkdir -p %{buildroot}/etc/niac
mkdir -p %{buildroot}/var/lib/niac
mkdir -p %{buildroot}/var/log/niac

# Copy binary
install -m 755 %{_repo_root}/niac %{buildroot}/usr/bin/niac

# Copy systemd service file
install -m 644 %{_repo_root}/deploy/systemd/niac.service %{buildroot}/usr/lib/systemd/system/niac.service

%files
%attr(755, root, root) /usr/bin/niac
%attr(644, root, root) /usr/lib/systemd/system/niac.service
%dir %attr(750, niac, niac) /etc/niac
%dir %attr(750, niac, niac) /var/lib/niac
%dir %attr(750, niac, niac) /var/log/niac

%pre
# Create service user and group
getent group niac >/dev/null || groupadd -r niac
getent passwd niac >/dev/null || \
    useradd -r -g niac -d /var/lib/niac -s /sbin/nologin \
    -c "NiAC Network Device Simulator" niac
exit 0

%post
# Set ownership of directories
chown -R niac:niac /etc/niac /var/lib/niac /var/log/niac

# Set capabilities for raw socket access
/usr/sbin/setcap 'cap_net_raw,cap_net_admin=+ep' /usr/bin/niac || true

# Configure firewall if firewalld is running
if systemctl is-active --quiet firewalld 2>/dev/null; then
    firewall-cmd --permanent --add-port=8080/tcp 2>/dev/null || true
    firewall-cmd --reload 2>/dev/null || true
    echo "Firewall configured for NiAC service (port 8080)"
fi

%systemd_post niac.service

%preun
%systemd_preun niac.service

%postun
%systemd_postun_with_restart niac.service

# On complete removal (not upgrade), clean up
if [ $1 -eq 0 ]; then
    # Remove firewall rules
    if systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --permanent --remove-port=8080/tcp 2>/dev/null || true
        firewall-cmd --reload 2>/dev/null || true
    fi

    # Remove user/group
    userdel niac 2>/dev/null || true
    groupdel niac 2>/dev/null || true
fi

%changelog
* Fri Jan 24 2025 Kris Armstrong <kris@mustardseednetworks.com>
- Added firewalld integration for automatic port configuration
- Added user/group Provides for Fedora compatibility

* Wed Jan 22 2025 Kris Armstrong <kris@mustardseednetworks.com>
- Initial RPM packaging
- Added systemd service with security hardening
- Added network capabilities for raw socket access
