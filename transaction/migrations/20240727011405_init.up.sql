CREATE TABLE IF NOT EXISTS "user" (
  pk serial NOT NULL PRIMARY KEY,
  user_id uuid UNIQUE NOT NULL,
  balance bigint NOT NULL,
  created_at timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "transaction" (
  pk serial NOT NULL PRIMARY KEY,
  user_id uuid NOT NULL,
  amount bigint NOT NULL,
  idempotency_key uuid UNIQUE NOT NULL,
  processed bool,
  event_timestamp timestamp NOT NULL
);
