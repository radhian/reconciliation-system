# Makefile

run:
	docker compose up --build

down:
	docker compose down -v

restart:
	docker compose down -v && docker compose up --build
