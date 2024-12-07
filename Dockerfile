# Pull the general AlmaLimux 9.4 image. Be sure to check
# https://hub.docker.com/_/almalinux for more info!
FROM almalinux:9.4

# Add handy metadata
LABEL \
    org.opencontainers.image.vendor="SciTags Organization"                                \
    org.opencontainers.image.source="https://github.com/scitags/flowd-go"                 \
    org.opencontainers.image.title="flowd-go-cont"                                        \
    org.opencontainers.image.description="Container for developing and building flowd-go" \
    org.opencontainers.image.authors="Pablo Collado Soto"
    # org.opencontainers.image.created=$BUILD_DATE                                        \
    # org.opencontainers.image.version=$BUILD_VERSION

# Versions of manually-installed tools
ARG GO_VERSION=1.23.0
ARG GH_CLI_VERSION=2.59.0

# Enable the CRB repo: libbpf-{devel,static} belong to it.
RUN dnf install -y epel-release && dnf config-manager --set-enabled crb

# Get YUM dependencies. Note 'yum check-update' returns 100 if updates are
# available, hence the trailing '|| true' to prevent it from failing.
RUN yum check-update || true  && \
    yum install -y \
        libbpf-devel-2:1.3.0  \
        libbpf-static-2:1.3.0 \
        clang-17.0.6          \
        llvm-17.0.6           \
        bpftool-7.3.0         \
        pandoc-2.14.0.3       \
        rpm-build-4.16.1.3    \
        rpm-devel-4.16.1.3    \
        rpmlint-1.11          \
        rpmdevtools-9.5       \
        make-1:4.3            \
        git-2.43.5            \
        tcpdump-4.99.0        \
        iproute-6.2.0         \
        iproute-tc-6.2.0      \
    && yum clean all

# Get Go
RUN curl -o go.tar.gz https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm -rf go.tar.gz

# Update the path so as to include Go
ENV PATH="/usr/local/go/bin:${PATH}"

# Install GH CLI for making releases in an easier fashion
RUN curl -LO https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz && \
    tar -xzf gh_${GH_CLI_VERSION}_linux_amd64.tar.gz && \
    cp gh_${GH_CLI_VERSION}_linux_amd64/bin/gh /bin && \
    rm -rf gh_${GH_CLI_VERSION}_linux_amd64*

# Setup the RPM build tree
RUN rpmdev-setuptree

# TODO: Consider compiling a given version of libbpf to link against
# TODO: a explained on https://github.com/libbpf/libbpf

# Move to the home directory
WORKDIR /root
