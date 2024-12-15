/*
 * Creates tables for managing agent registration vouchers
 *
 * Tables:
 * - vouchers: Stores active voucher information
 * - temp_vouchers: Manages temporary vouchers during creation
 *
 * Security:
 * - Unique constraints prevent duplicate codes
 * - Foreign key constraints maintain data integrity
 * - Code format validation ensures consistent voucher format
 *
 * Indexes:
 * - idx_vouchers_code: Optimizes voucher code lookups
 * - idx_temp_vouchers_code: Improves temporary voucher queries
 */

CREATE TABLE vouchers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(12) NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id),
    is_continuous BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    disabled_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT valid_code CHECK (code ~ '^[A-Z0-9]{12}$')
);

CREATE TABLE temp_vouchers (
    code VARCHAR(12) NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id),
    is_continuous BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT valid_code CHECK (code ~ '^[A-Z0-9]{12}$')
);

-- Index for faster lookups
CREATE INDEX idx_vouchers_code ON vouchers(code);
CREATE INDEX idx_temp_vouchers_code ON temp_vouchers(code);

-- Add comments for better documentation
COMMENT ON TABLE vouchers IS 'Stores active agent registration vouchers';
COMMENT ON TABLE temp_vouchers IS 'Manages temporary vouchers during creation process';
COMMENT ON COLUMN vouchers.is_continuous IS 'Indicates if voucher can be used multiple times';
COMMENT ON COLUMN temp_vouchers.expires_at IS 'Timestamp when temporary voucher becomes invalid';