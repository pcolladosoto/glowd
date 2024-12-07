# This Makefile provides several targets hadling the creation of of RPM packages
# and documentation for publishing flowd-go

# The directory on which to place the build files and resulting RPMs for easier access
RPM_DIR = rpms

# Markdown-formatted manpage to parse with pandoc.
DOC_FILE = $(BIN_NAME).1.md

# Path of the buildroot created with rpmdev-setuptree. In order to create
# it, install the necessary tools as explained on the main README.md and
# then simply run rpmdev-setuptree. The build tree will be created by
# default on ${HOME}/rpmbuild, hence the definition of this variable.
RPM_BUILDROOT = $(shell echo ${HOME})/rpmbuild

# Be sure to check https://rpm-packaging-guide.github.io for more info!
rpm: doc build $(RPM_BUILDROOT)
	@echo "Building RPM with buildroot $(RPM_BUILDROOT)"
	@echo "Copying artifacts to the RPM buildroot..."
	@cp $(BIN_DIR)/$(BIN_NAME)                $(RPM_BUILDROOT)/SOURCES/$(BIN_NAME)
	@cp $(RPM_DIR)/conf.json                  $(RPM_BUILDROOT)/SOURCES/$(BIN_NAME).json
	@cp $(RPM_DIR)/$(BIN_NAME).service        $(RPM_BUILDROOT)/SOURCES/$(BIN_NAME).service
	@cp $(RPM_DIR)/$(basename $(DOC_FILE)).gz $(RPM_BUILDROOT)/SOURCES/$(basename $(DOC_FILE)).gz
	@echo "Building the RPM..."
	@rpmbuild -bb $(RPM_DIR)/$(BIN_NAME).spec
	@echo "Copying the RPM to $(RPM_DIR)"
	@cp $(RPM_BUILDROOT)/RPMS/x86_64/flowd-go-$(VERSION)-1.x86_64.rpm $(RPM_DIR)

# If the devtree doesn't exist (i.e. we're running in the CI container) simply create it
$(RPM_BUILDROOT):
	@echo "Creating RPM build tree..."
	rpmdev-setuptree

doc: $(RPM_DIR)/$(DOC_FILE)
	@echo "Building documentation"
	@pandoc --standalone --to man $(RPM_DIR)/$(DOC_FILE) | gzip > $(RPM_DIR)/$(basename $(DOC_FILE)).gz
