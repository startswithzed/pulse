.PHONY: build build-no-cache up down clean logs restart

build:
	docker compose build

build-no-cache:
	docker compose build --no-cache

up:
	docker compose up --attach pulse-gateway --attach pulse-engine

down:
	docker compose down

logs:
	docker compose logs -f pulse-gateway pulse-engine

restart:
	docker compose restart pulse-gateway pulse-engine

clean:
	docker compose down -v
	rm -rf apps/gateway/tmp apps/engine/tmp
