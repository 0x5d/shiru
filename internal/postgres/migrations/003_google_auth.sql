ALTER TABLE users
  ADD COLUMN google_sub    TEXT,
  ADD COLUMN email         TEXT,
  ADD COLUMN name          TEXT,
  ADD COLUMN avatar_url    TEXT,
  ADD COLUMN last_login_at TIMESTAMPTZ;

ALTER TABLE users
  ADD CONSTRAINT users_google_sub_nonempty CHECK (google_sub IS NULL OR google_sub <> ''),
  ADD CONSTRAINT users_email_nonempty CHECK (email IS NULL OR email <> '');

CREATE UNIQUE INDEX users_google_sub_unique ON users (google_sub) WHERE google_sub IS NOT NULL;
