
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

-- +goose StatementBegin
DO $$
BEGIN
IF EXISTS(
    SELECT env
      FROM development.platform
     WHERE env = 'development'
)
THEN

-- User for 'Frodo'

INSERT INTO users(
    user_id, email, gender, language, created,
    state, password, password_date, dob_day,
    dob_month, dob_year
) VALUES (
    1, 'frodo@microcosm.cc', NULL, 'en-gb', NOW(),
    'email_confirm', 'password_hash', NOW(), NULL,
    NULL, NULL
);
PERFORM pg_catalog.setval('users_user_id_seq', 1, true);

-- Profile pic for 'Frodo'

INSERT INTO attachment_meta(
    created, file_size, file_sha1, mime_type, width,
    height, thumbnail_width, thumbnail_height, attach_count
) VALUES (
    NOW(), 19566 , '66cca61feb8001cb71a9fb7062ff94c9d2543340', 'image/png', 96,
    96, 0, 0, 3
);
PERFORM pg_catalog.setval('attachment_meta_attachment_meta_id_seq', 1, true);

-- Site: root, site_id 1 must always be the root site that is the management site
-- for creating and editing other sites

-- THIS SITE IS REQUIRED IN ALL MICROCOSM INSTANCES, PRODUCTION AND DEV ENVS

PERFORM create_owned_site(
           'Root', -- site title
           'root', -- subdomain key
           1, --theme id
           1, -- user id
           'Frodo', -- profile name
           NULL, -- avatar id
           '/api/v1/files/66cca61feb8001cb71a9fb7062ff94c9d2543340', -- avatar url
           'root.localhost', -- custom domain
           'Root site for creating and managing sites', -- site description
           NULL, -- logo url
           NULL, -- background url
           'tile', -- background position
           '#FFFFFF', -- background color
           '#4082C3', -- link color
           NULL -- google analytics code
       );

-- Site: localhost, a Microcosm site

PERFORM create_owned_site(
           'localhost', -- site title
           'localhost', -- subdomain key
           1, --theme id
           1, -- user id
           'Frodo', -- profile name
           NULL, -- avatar id
           '/api/v1/files/66cca61feb8001cb71a9fb7062ff94c9d2543340', -- avatar url
           'localhost', -- custom domain
           'localhost', -- site description
           NULL, -- logo url
           NULL, -- background url
           'tile', -- background position
           '#FFFFFF', -- background color
           '#4082C3', -- link color
           NULL -- google analytics code
       );

-- Site: dev1, a Microcosm development site

PERFORM create_owned_site(
           'dev1', -- site title
           'dev1', -- subdomain key
           1, --theme id
           1, -- user id
           'Frodo', -- profile name
           NULL, -- avatar id
           '/api/v1/files/66cca61feb8001cb71a9fb7062ff94c9d2543340', -- avatar url
           'dev1.localhost', -- custom domain
           'dev1', -- site description
           NULL, -- logo url
           NULL, -- background url
           'tile', -- background position
           '#FFFFFF', -- background color
           '#4082C3', -- link color
           NULL -- google analytics code
       );

-- Site: dev2, a Microcosm development site

PERFORM create_owned_site(
           'dev2', -- site title
           'dev2', -- subdomain key
           1, --theme id
           1, -- user id
           'Frodo', -- profile name
           NULL, -- avatar id
           '/api/v1/files/66cca61feb8001cb71a9fb7062ff94c9d2543340', -- avatar url
           'dev2.localhost', -- custom domain
           'dev2', -- site description
           NULL, -- logo url
           NULL, -- background url
           'tile', -- background position
           '#FFFFFF', -- background color
           '#4082C3', -- link color
           NULL -- google analytics code
       );

-- Default OAuth client

    INSERT INTO oauth_clients (
        client_id, name, created, client_secret
    ) VALUES (
        1, 'microweb', NOW(), 'CHANGE THIS STRING TO BE A RANDOM SECRET FOR THIS CLIENT'
    );
    PERFORM pg_catalog.setval('oauth_clients_client_id_seq', 1, true);

    INSERT INTO access_tokens(
        token_value, user_id, client_id
    ) VALUES (
        'letmein', 1, 1
    );

END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;
