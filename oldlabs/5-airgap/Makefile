CHANNEL=$(shell basename $(shell pwd))

release: 
	@replicated release create \
		--auto -y \
		--promote $(CHANNEL)
