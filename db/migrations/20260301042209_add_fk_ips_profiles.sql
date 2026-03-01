-- +goose Up
ALTER TABLE ips
ADD CONSTRAINT fk_ips_profile_id
FOREIGN KEY (profile_id)
REFERENCES profiles (profile_id)
ON DELETE CASCADE;

-- +goose Down
ALTER TABLE ips
DROP CONSTRAINT fk_ips_profile_id;
