
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION canonical_email(
	email varchar
)
  RETURNS varchar AS
$BODY$
DECLARE
canonicalised varchar;
localpart varchar;
domainpart varchar;
BEGIN

-- Important to note that this function does not validate an email address, it
-- merely produces a canonical form of the email address.

-- lowercase
canonicalised = LOWER(email);

-- localpart is the user@ part
localpart = SPLIT_PART(canonicalised, '@', 1);

-- ignore dots in the user part
localpart = REPLACE(localpart, '.', '');

-- ignore everything after + in the user part
localpart = SPLIT_PART(localpart, '+', 1);

-- strip comment prefix
IF localpart ~ '^\([^(]*\).+' THEN
	localpart = regexp_replace(localpart, '^\([^(]*\)', '');
END IF;

-- strip comment suffix
IF localpart ~ '.+\([^(]*\)$' THEN
	localpart = regexp_replace(localpart, '\([^(]*\)$', '');
END IF;

-- domainpart is the @domain part
domainpart = SPLIT_PART(canonicalised, '@', 2);

CASE domainpart

-- Apple
WHEN 'mac.com' THEN
	domainpart = 'icloud.com';
WHEN 'me.com' THEN
	domainpart = 'icloud.com';

-- BT
WHEN 'btopenworld.com' THEN
	domainpart = 'btinternet.com';

-- GMX
WHEN 'gmx.at' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.ch' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.com' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.co.uk' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.fr' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.li' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.net' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.org' THEN
	domainpart = 'gmx.de';
WHEN 'gmx.us' THEN
	domainpart = 'gmx.de';

-- Google
WHEN 'googlemail.com' THEN
	domainpart = 'gmail.com';

-- Outlook
WHEN 'hotmail.be' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.co.jp' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.com' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.com.au' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.com.tw' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.con' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.co.nz' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.co.uk' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.de' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.es' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.fi' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.fr' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.it' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.lt' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.lv' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.nl' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.no' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.org' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.se' THEN
	domainpart = 'outlook.com';
WHEN 'hotmail.sg' THEN
	domainpart = 'outlook.com';
WHEN 'live.com' THEN
	domainpart = 'outlook.com';
WHEN 'live.co.uk' THEN
	domainpart = 'outlook.com';
WHEN 'msn.com' THEN
	domainpart = 'outlook.com';
WHEN 'passport.com' THEN
	domainpart = 'outlook.com';

--Virgin
WHEN 'virgin.net' THEN
	domainpart = 'virginmedia.com';

-- Yahoo
WHEN 'yahoo.ca' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.co.id' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.co.in' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.co.jp' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.ar' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.au' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.br' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.cn' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.hk' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.mx' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.my' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.ph' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.sg' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.tw' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.com.uk' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.co.nz' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.co.uk' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.de' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.dk' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.es' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.fr' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.gr' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.ie' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.it' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoomail.com' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.no' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.pl' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.ro' THEN
	domainpart = 'yahoo.com';
WHEN 'yahoo.se' THEN
	domainpart = 'yahoo.com';

ELSE

END CASE;

canonicalised = localpart || '@' || domainpart;

RETURN canonicalised;
END;
$BODY$
  LANGUAGE 'plpgsql' STABLE
  COST 100;
-- +goose StatementEnd

ALTER FUNCTION canonical_email(varchar) OWNER TO microcosm;

ALTER TABLE users ADD COLUMN canonical_email character varying(254);

COMMENT ON COLUMN users.canonical_email IS 'The canonical representation of an email for the purpose of uniquely identifying accounts.

The email column is the address to which we send email (and is the first version of the email we see for a customer), whereas the canonical email is the address by which we identify a user.';

UPDATE users u
   SET canonical_email = canonical_email(email)
 WHERE canonical_email IS NULL;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE users DROP COLUMN canonical_email;
DROP FUNCTION canonical_email(varchar);
