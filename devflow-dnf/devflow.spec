%global debug_package %{nil}

Name:           devflow
Version:        0.1.0
Release:        1%{?dist}
Summary:        Lightweight workflow orchestrator for local development

License:        MIT
URL:            https://github.com/anandyadav3559/devflow
Source0:        https://github.com/anandyadav3559/devflow/archive/refs/heads/main.tar.gz

BuildRequires:  golang >= 1.21

%description
DevFlow is a local development environment orchestrator. It lets you define
multi-service workflows in YAML and run them with dependency management,
logging, and easy reuse.

%prep
%autosetup -n %{name}-main

%build
# Set standard Go build flags for Fedora
export CGO_CPPFLAGS="${CPPFLAGS}"
export CGO_CFLAGS="${CFLAGS}"
export CGO_CXXFLAGS="${CXXFLAGS}"
export CGO_LDFLAGS="${LDFLAGS}"
export GOFLAGS="-buildmode=pie -trimpath -mod=vendor -modcacherw"

go build -ldflags="-s -w -linkmode=external" -o %{name} .

%install
# Install the compiled binary into /usr/bin
install -Dm755 %{name} %{buildroot}%{_bindir}/%{name}

%files
%doc README.md
%{_bindir}/%{name}

%changelog
* Tue Mar 24 2026 Anand Yadav - 0.1.0-1
- Initial COPR release
