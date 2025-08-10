DB_URL=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable

network:
	docker network create jss-network

postgres:
	docker run --name postgres --network jss-network -p 9999:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root jss_product

dropdb:
	docker exec -it postgres dropdb jss_product

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)