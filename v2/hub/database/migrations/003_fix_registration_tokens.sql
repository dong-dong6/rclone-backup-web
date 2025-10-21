-- Add missing 'used' column to registration_tokens table if it doesn't exist
ALTER TABLE registration_tokens 
ADD COLUMN IF NOT EXISTS used BOOLEAN DEFAULT FALSE NOT NULL;

-- Update the 'used' field based on 'used_at'
UPDATE registration_tokens 
SET used = TRUE 
WHERE used_at IS NOT NULL;

-- Create a function to automatically set 'used_at' when 'used' is set to true
CREATE OR REPLACE FUNCTION update_used_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.used = TRUE AND OLD.used = FALSE AND NEW.used_at IS NULL THEN
        NEW.used_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to update used_at when used is set to true
DROP TRIGGER IF EXISTS update_registration_token_used_at ON registration_tokens;
CREATE TRIGGER update_registration_token_used_at
    BEFORE UPDATE ON registration_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_used_at();