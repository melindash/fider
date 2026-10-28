-- Mage-OS fork. Votes carried over from GitHub discussions, kept apart from votes cast on the board.
ALTER TABLE posts ADD COLUMN IF NOT EXISTS mageos_historical_votes INT NOT NULL DEFAULT 0;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS mageos_historical_source TEXT NULL;
