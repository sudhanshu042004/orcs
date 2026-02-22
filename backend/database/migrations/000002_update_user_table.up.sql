ALTER TABLE users
ADD username TEXT,
ADD email TEXT UNIQUE,
ADD avatar TEXT,
ADD repo_url TEXT NOT NULL;