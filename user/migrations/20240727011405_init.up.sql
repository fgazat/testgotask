CREATE TABLE "user" (
  pk serial NOT NULL PRIMARY KEY,
  user_id uuid UNIQUE NOT NULL,
  email VARCHAR(70) UNIQUE NOT NULL,
  created_at timestamp NOT NULL DEFAULT NOW ()
);
