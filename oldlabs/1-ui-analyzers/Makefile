CHANNEL=$(shell basename $(shell pwd))

release: 
	@replicated release create \
		--app ${REPLICATED_APP} \
		--token ${REPLICATED_API_TOKEN} \
		--auto -y \
		--promote $(CHANNEL)
