CREATE TABLE "user" (
  user_id uuid PRIMARY KEY,
  email VARCHAR(70) UNIQUE NOT NULL,
  created_at timestamp NOT NULL DEFAULT NOW ()
);
