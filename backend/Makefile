CONF ?= configs/config.local.yaml

.PHONY: init-config run migrate seed init-admin build test wire
init-config:
	@test -f "$(CONF)" || (umask 077; cp configs/config.local.example.yaml "$(CONF)")
run:
	go run ./cmd/server -conf "$(CONF)"
migrate:
	go run ./cmd/server -conf "$(CONF)" -migrate
seed:
	go run ./cmd/server -conf "$(CONF)" -seed
init-admin:
	go run ./cmd/server -conf "$(CONF)" -init-admin
build:
	go build -o bin/server ./cmd/server
test:
	go test ./...
wire:
	go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server
