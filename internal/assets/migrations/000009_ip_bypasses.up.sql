CREATE TABLE IF NOT EXISTS "ip_bypasses" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "cidr" TEXT NOT NULL,
    "domain" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "note" TEXT NOT NULL DEFAULT '',
    "created_by" TEXT NOT NULL DEFAULT '',
    "created_at" INTEGER NOT NULL DEFAULT 0
);
