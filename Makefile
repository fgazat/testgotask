install:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	
rebuild:
	docker-compose up --build -d


