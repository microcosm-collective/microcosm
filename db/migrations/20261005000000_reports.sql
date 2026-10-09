-- +goose Up
-- +goose StatementBegin
CREATE TABLE report_types (
    report_type_id bigserial PRIMARY KEY,
    key varchar(64) NOT NULL UNIQUE,
    title varchar(128) NOT NULL,
    sort_order smallint NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO report_types (key, title, sort_order) VALUES
    ('hate_speech',      'Hate speech',                  1),
    ('nudity',           'Nudity or sexual activity',    2),
    ('harassment',       'Harassment or bullying',       3),
    ('illegal_activity', 'Illegal activity',             4),
    ('spam',             'Spam',                         5),
    ('fraud',            'Scam, fraud or impersonation', 6),
    ('other',            'Other',                        7);

CREATE TABLE reports (
    report_id bigserial PRIMARY KEY,
    site_id bigint NOT NULL REFERENCES sites(site_id),
    comment_id bigint NOT NULL UNIQUE REFERENCES comments(comment_id),
    state varchar(16) NOT NULL DEFAULT 'open'
        CHECK (state IN ('open', 'reviewing', 'closed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    closed_at timestamptz
);

CREATE TABLE reporters (
    reporter_id bigserial PRIMARY KEY,
    report_id bigint NOT NULL REFERENCES reports(report_id) ON DELETE CASCADE,
    profile_id bigint NOT NULL REFERENCES profiles(profile_id),
    report_type_id bigint NOT NULL REFERENCES report_types(report_type_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (report_id, profile_id)
);

CREATE TABLE report_state_changes (
    report_state_change_id bigserial PRIMARY KEY,
    report_id bigint NOT NULL REFERENCES reports(report_id) ON DELETE CASCADE,
    changed_by_profile_id bigint NOT NULL REFERENCES profiles(profile_id),
    from_state varchar(16) CHECK (from_state IN ('open', 'reviewing', 'closed')),
    to_state varchar(16) NOT NULL CHECK (to_state IN ('open', 'reviewing', 'closed')),
    note text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX reports_site_state_created_idx ON reports (site_id, state, created_at DESC);
CREATE INDEX reporters_report_created_idx ON reporters (report_id, created_at);
CREATE INDEX report_state_changes_report_created_idx ON report_state_changes (report_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE report_state_changes;
DROP TABLE reporters;
DROP TABLE reports;
DROP TABLE report_types;
-- +goose StatementEnd
