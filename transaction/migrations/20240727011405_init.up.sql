CREATE TABLE IF NOT EXISTS "user" (
  pk serial NOT NULL PRIMARY KEY,
  user_id uuid UNIQUE NOT NULL,
  balance BIGINT NOT NULL,
  created_at timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "transaction" (
  pk serial NOT NULL PRIMARY KEY,
  user_id uuid UNIQUE NOT NULL,
  amount BIGINT NOT NULL,
  event_timestamp timestamp NOT NULL
);
