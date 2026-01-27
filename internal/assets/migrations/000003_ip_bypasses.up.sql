CREATE TABLE IF NOT EXISTS "ip_bypasses" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "cidr" TEXT NOT NULL,
    "domain" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "note" TEXT,
    "created_at" INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_ip_bypasses_cidr_domain ON ip_bypasses(cidr, domain);
CREATE INDEX idx_ip_bypasses_domain_expires ON ip_bypasses(domain, expires_at);
CREATE INDEX idx_ip_bypasses_expires ON ip_bypasses(expires_at);
