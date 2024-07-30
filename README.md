# testgotask

## Local development

```bash
# 1. Start docker container
docker-compose up --build -d

# 2. Install migrate CLI.
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 3. Apply migrations for user db.
migrate -source file://./user/migrations -database "postgres://admin:root@localhost:5432/user?sslmode=disable" up

# 4. Apply migrations for transaction db.
migrate -source file://./transaction/migrations -database "postgres://admin:root@localhost:5433/transaction?sslmode=disable" up

# curls 
# Create_user
curl -X POST 0.0.0.0:8081/create_user -d '{"email": "another@gmail.com"}'
curl -X POST 0.0.0.0:8081/create_user -d '{"email": "test@gmail.com"}'
# Balance
curl -X GET 0.0.0.0:8081/balance -d '{"email": "test@gmail.com"}'
# Add_money
curl -X POST 0.0.0.0:8082/add_money -d '{"user_id": "UUID", "amount": 14000}'
# Transfer_money
curl -X POST 0.0.0.0:8082/transfer_money -d '{"from_user_id": "UUID", "to_user_id": "UUID", "amount": 14000}'

# Connection strings
# user db
psql "host=localhost \
    port=5432 \
    sslmode=disable \
    dbname=user \
    user=admin \
    target_session_attrs=read-write"

# transaction db
psql "host=localhost \
    port=5433 \
    sslmode=disable \
    dbname=transaction \
    user=admin \
    target_session_attrs=read-write"
```

## TODO

* replace interactions between microservices with protobufs
* refactoring
