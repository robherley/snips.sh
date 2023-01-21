CREATE TABLE public_keys (
  fingerprint TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  user_id TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS public_keys_user_id ON public_keys (user_id);
