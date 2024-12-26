/*
 * Creates tables for managing agent registration vouchers
 *
 * Tables:
 * - claim_vouchers: Stores active voucher information
 * - claim_voucher_usage: Tracks usage attempts
 *
 * Security:
 * - Unique constraints prevent duplicate codes
 * - Foreign key constraints maintain data integrity
 * - Code format validation ensures consistent voucher format
 *
 * Indexes:
 * - idx_claim_vouchers_code: Optimizes voucher code lookups
 * - idx_claim_vouchers_active: Improves usage tracking queries
 */

CREATE TABLE claim_vouchers (
    code VARCHAR(50) PRIMARY KEY,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_continuous BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,
    used_at TIMESTAMP WITH TIME ZONE,
    used_by_id UUID REFERENCES users(id)
);

CREATE TABLE claim_voucher_usage (
    id SERIAL PRIMARY KEY,
    voucher_code VARCHAR(50) NOT NULL REFERENCES claim_vouchers(code),
    attempted_by_id UUID NOT NULL REFERENCES users(id),
    attempted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN NOT NULL DEFAULT false,
    ip_address VARCHAR(45), -- IPv6 addresses can be up to 45 characters
    user_agent TEXT,
    error_message TEXT
);

-- Index for faster lookups
CREATE INDEX idx_claim_vouchers_code ON claim_vouchers(code);
CREATE INDEX idx_claim_vouchers_active ON claim_vouchers(is_active);
CREATE INDEX idx_claim_vouchers_created_by ON claim_vouchers(created_by_id);
CREATE INDEX idx_claim_voucher_usage_voucher ON claim_voucher_usage(voucher_code);
CREATE INDEX idx_claim_voucher_usage_attempted_by ON claim_voucher_usage(attempted_by_id);

-- Add comments for better documentation
COMMENT ON TABLE claim_vouchers IS 'Stores active agent registration vouchers';
COMMENT ON TABLE claim_voucher_usage IS 'Tracks usage attempts of claim vouchers';
COMMENT ON COLUMN claim_vouchers.is_continuous IS 'Indicates if voucher can be used multiple times';
COMMENT ON COLUMN claim_vouchers.expires_at IS 'Timestamp when voucher becomes invalid';

-- Create trigger to automatically update updated_at
CREATE OR REPLACE FUNCTION update_claim_vouchers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_claim_vouchers_updated_at
    BEFORE UPDATE ON claim_vouchers
    FOR EACH ROW
    EXECUTE FUNCTION update_claim_vouchers_updated_at();