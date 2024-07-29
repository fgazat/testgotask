CREATE TABLE "user" (
  pk INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
  user_id uuid UNIQUE NOT NULL,
  balance BIGINT NOT NULL,
  created_at timestamp NOT NULL
);

CREATE TABLE "transaction" (
  pk INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
  user_id uuid UNIQUE NOT NULL,
  amount BIGINT NOT NULL,
  event_timestamp timestamp NOT NULL
);
