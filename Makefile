postgres:
	docker run --name postgres12 --network bank-network -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres12 dropdb simple_bank

enterdb:
	docker exec -it postgres12 psql -U root simple_bank

migrateup:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5433/simple_bank?sslmode=disable" -verbose up $(n)

migratedown:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5433/simple_bank?sslmode=disable" -verbose down $(n)

sqlc:
	sqlc generate

mock:
	mockgen -destination db/mock/store.go -package mockdb github.com/Andrew-2609/simple-bank/db/sqlc Store

dockerup:
	docker-compose up

dockerdown:
	docker-compose down

testlocal:
	go clean -testcache && grc go test -v -cover ./...

test:
	go clean -testcache && go test -v -cover ./...

up:
	go run main.go

.PHONY: postgres createdb dropdb enterdb migrateup migratedown sqlc mock dockerup dockerdown testlocal test up
