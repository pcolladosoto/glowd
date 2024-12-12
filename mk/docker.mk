# This Makefile provides several PHONY targets automating the creation, deletion
# and use of the auxiliary container for development and debugging.

# Get a hold of the current directory so as to mount the volume with an absolute path.
# Otherwise Docker will try to create its own volume, at least on Docker Desktop-based
# platforms.
PWD = $(shell pwd)

# Unclutter the target recipes a bit by pulling information out.
CONTAINER_IMAGE = ghcr.io/scitags/flowd-go:v1.0
CONTAINER_NAME = flowd-go

# Now, what do each of these flags do?
#                       -d: Run the container in the background. It'll be kept alive by the bash session we trigger.
#                       -t: Allocate a pseudo-TTY so that bash doesn't exit as soon as it's launched. Otherwise, the conatiner would quit and
#                           we'd have nothing to attach to later!
#                     --rm: Delete the container once we exit to help with the cleanup.
# -v $(PWD):/root/flowd-go: This will mount the current directory (i.e. the flowd-go repo) under /root/flowd-go in the container.
#      --cap-add SYS_ADMIN: Add the CAP_SYS_ADMIN capability to the container which allows us to do a bunch of stuff. Check capabilities(7).
#            --cap-add BPF: Add the CAP_BPF capability which, unsurprisingly, allows loading BPF programs into the kernel. Bear in mind
#                           this capability exists since Linux 5.8: it was part of CAP_SYS_ADMIN before!
#      --cap-add NET_ADMIN: Add the CAP_NET_ADMIN capability, allowing us to create the qdiscs to attach the BPF programs to.
# --name $(CONTAINER_NAME): Give the container a deterministic name (i.e. flowd-go) so that the following commands are reproducible.
#       $(CONTAINER_IMAGE): The purposefully built image we are going to run.
#                     bash: The command to run (i.e. a regular bash shell).
# Now, we'll also mount the debugfs filesyste automatically so that we don't rely on the users doing it themselves and/or
# the user creating a persistent volume beforehand with:
#     $ docker volume create --driver local --opt type=debugfs --opt device=debugfs debugfs
# At any rate, check https://hemslo.io/run-ebpf-programs-in-docker-using-docker-bpf/ for a bit more context
.PHONY: docker-start
docker-start:
	@docker run -d -t --rm -v $(PWD):/root/flowd-go \
		--cap-add SYS_ADMIN --cap-add BPF --cap-add NET_ADMIN \
		--name $(CONTAINER_NAME) $(CONTAINER_IMAGE)

	@docker exec -it $(CONTAINER_NAME) mount -t debugfs debugfs /sys/kernel/debug

# Simply run a shell within the container. Bear in mind the trailing '|| true' simply
# avoids having make print errors when the shell is not closed cleanly.
.PHONY: docker-shell
docker-shell:
	@docker exec -it $(CONTAINER_NAME) bash || true

# Stop the container: as we passed the --rm flag to run it'll be deleted automatically.
# We 'force' the termination by passing SIGKILL so that the initial bash shell spawned
# with 'docker run ...' terminates right away: the state it's on is not quite clean...
.PHONY: docker-stop
docker-stop:
	@docker stop -s SIGKILL $(CONTAINER_NAME)

# Please note we should handle the tag in an automatic way, maybe making use of constructs
# such as '$(shell git describe --tags --abbrev=0)' or something of the sort. That's on
# the TODO list!
.PHONY: docker-build
docker-build:
	@docker build -t $(CONTAINER_IMAGE) .

# Push the image to GitHub's registry.
.PHONY: docker-push
docker-push:
	@docker push $(CONTAINER_IMAGE)
