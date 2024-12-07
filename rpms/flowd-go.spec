Name:		flowd-go
Version:	2.0
Release:	1
Summary:	SciTags Flowd-go Daemon
BuildArch:	x86_64

URL: https://github.com/scitags/flowd-go

License:	GPLv3

BuildRequires:	systemd

# Longer description on what the package is/does
%description
Reimplementation of the flowd daemon in Go.

The flowd-go daemon serves as the backbone for HEPiX's
SciTags initiative as seen on https://www.scitags.org

# Time to copy the binary file!
%install
# Delete the previos build root
rm -rf %{buildroot}

# Create the necessary directories
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_unitdir}
mkdir -p %{buildroot}%{_sysconfdir}/%{name}
mkdir -p %{buildroot}%{_mandir}/man1

# And install the necessary files
install -m 0775 %{_sourcedir}/%{name}         %{buildroot}%{_bindir}/%{name}
install -m 0644 %{_sourcedir}/%{name}.json    %{buildroot}%{_sysconfdir}/%{name}/conf.json
install -m 0664 %{_sourcedir}/%{name}.service %{buildroot}%{_unitdir}/%{name}.service
install -m 0664 %{_sourcedir}/%{name}.1.gz    %{buildroot}%{_mandir}/man1/%{name}.1.gz

# Files provided by the package. Check https://docs.fedoraproject.org/en-US/packaging-guidelines/#_manpages too!
%files
%{_bindir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}/conf.json
%{_unitdir}/%{name}.service
%{_mandir}/man1/%{name}.1*

# Changes introducd with each version
%changelog
* Thu Nov 14 2024 Pablo Collado Soto <pablo.collado.soto@cern.ch> - 2.0-1
- Update to flowd-go-2.0

* Wed Nov 6 2024 Pablo Collado Soto <pablo.collado.soto@cern.ch> - 1.0-1
- Release flowd-go-1.0
