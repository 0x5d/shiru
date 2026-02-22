ALTER TABLE users
  ADD COLUMN google_sub    TEXT,
  ADD COLUMN email         TEXT,
  ADD COLUMN name          TEXT,
  ADD COLUMN avatar_url    TEXT,
  ADD COLUMN last_login_at TIMESTAMPTZ;

CREATE UNIQUE INDEX users_google_sub_unique ON users (google_sub) WHERE google_sub IS NOT NULL;
