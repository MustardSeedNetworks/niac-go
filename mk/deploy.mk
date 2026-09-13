# =============================================================================
# Deployment validation
# =============================================================================
# Packages are built by goreleaser in CI, not here — there is deliberately no
# local `deb`/`rpm` target duplicating that path. What the build contract needs
# locally is the check that runs AFTER an install: that the service answering
# on the host is the artifact the release published.
#
#   make deploy-validate HOST=dev-srv-ubuntu
#   make deploy-validate HOST=dev-srv-fedora RELEASE=v0.95.38
#   make deploy-validate HOST=... PACKAGE=./dist/niac_0.95.38_amd64.deb
#   make deploy-validate HOST=... FROM_RELEASE=v0.95.57   # or FROM_RELEASE=none
#
# HOST is an ssh target and is required; the deployment hosts are supplied by
# variable rather than baked in, because the last set of hardcoded names went
# stale when the lab re-addressed. RELEASE, not VERSION: mk/vars.mk already
# owns VERSION as this working tree's `git describe`, which is not a release.
#
# FROM_RELEASE names the build installed FIRST, so the second install is a real
# version-to-version upgrade rather than a reinstall of the same bytes. It
# defaults to the release immediately before RELEASE; `none` restores the
# same-version reinstall.
# =============================================================================

.PHONY: deploy-validate

deploy-validate: ## Install a released package on HOST and validate /__version + upgrade
ifndef HOST
	$(error HOST is required, e.g. make deploy-validate HOST=dev-srv-ubuntu)
endif
	@scripts/lab/deploy-validate.sh --host $(HOST) \
		$(if $(RELEASE),--version $(RELEASE)) \
		$(if $(FROM_RELEASE),--from-version $(FROM_RELEASE)) \
		$(if $(PORT),--port $(PORT)) \
		$(if $(PACKAGE),--package $(PACKAGE))
