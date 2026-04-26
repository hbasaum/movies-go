include .envrc

# Create th new confirm target
confirm:
	@echo -n 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y ]

.PHONY: run/api
run/api:
	go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN} -smtp-username=01f8fb979f4ade -smtp-password=5ab0b604cc06a2	

db/psql:
	psql ${GREENLIGHT_DB_DSN}

db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path ./migrations -database ${GREENLIGHT_DB_DSN} up

.PHONY: build/api
build/api:
	@echo 'building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -o=./bin/linux_amd64/api ./cmd/api

# ==================================================================================== #
# PRODUCTION
# ==================================================================================== #

production_host = greenlight@141.148.195.43
production_ssh_key = ~/.ssh/orc-amd.key

## production/connect: connect to the production server
.PHONY: production/connect
production/connect:
	ssh -i $(production_ssh_key) $(production_host)

## production/deploy/api: deploy the api to production
.PHONY: production/deploy/api
production/deploy/api:
	rsync -P -e "ssh -i $(production_ssh_key)" ./bin/linux_amd64/api $(production_host):~
	rsync -rP --delete -e "ssh -i $(production_ssh_key)" ./migrations $(production_host):~
	ssh -i $(production_ssh_key) -t $(production_host) 'migrate -path ~/migrations -database $$GREENLIGHT_DB_DSN up'
