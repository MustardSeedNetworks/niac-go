-- MIB Database Schema for NIAC-Go
-- Stores OID name mappings and VariMib handler configurations

-- OID name-to-numeric mappings
-- Ported from Java OidMap.java (918+ entries)
CREATE TABLE IF NOT EXISTS oid_names (
    name TEXT PRIMARY KEY,           -- Human-readable name (e.g., "sysDescr")
    oid TEXT NOT NULL,               -- Numeric OID (e.g., "1.3.6.1.2.1.1.1")
    full_path TEXT,                  -- Full descriptive path (optional)
    mib_name TEXT,                   -- Source MIB name (e.g., "SNMPv2-MIB")
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for OID prefix searches
CREATE INDEX IF NOT EXISTS idx_oid_names_oid ON oid_names(oid);

-- VariMib handler configurations
-- Supports time-varying counters and string cycling from Java Handler.java
CREATE TABLE IF NOT EXISTS varimib_handlers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    oid_pattern TEXT NOT NULL,       -- Exact OID or pattern with wildcards
    handler_type TEXT NOT NULL,      -- "integral", "string", "fixed", "sysuptime"
    config TEXT NOT NULL,            -- Handler-specific configuration
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(oid_pattern)
);

-- Walk file cache for loaded SNMP walk data
CREATE TABLE IF NOT EXISTS walk_cache (
    oid TEXT PRIMARY KEY,
    asn_type TEXT NOT NULL,          -- ASN.1 type (INTEGER, Counter32, etc.)
    value TEXT NOT NULL,             -- Stored value
    source_file TEXT,                -- Source walk file path
    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for walk cache OID lookups
CREATE INDEX IF NOT EXISTS idx_walk_cache_oid ON walk_cache(oid);

-- MIB source tracking for documentation
CREATE TABLE IF NOT EXISTS mib_sources (
    mib_name TEXT PRIMARY KEY,
    description TEXT,
    vendor TEXT,
    rfc_reference TEXT,
    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
