-- +goose Up
-- +goose StatementBegin

CREATE TABLE report_reasons (
	report_reason_id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	title text NOT NULL UNIQUE,
	description text DEFAULT '' NOT NULL
);

INSERT INTO report_reasons (title, description)
VALUES
    ('Hate Speech', 'Content that promotes hatred against protected groups'),
    ('Harassment', 'Content that harasses, intimidates, or bullies others'),
    ('Misinformation', 'Deliberately false or misleading content'),
    ('Illegal Content', 'Content that violates local, state, or federal laws'),
    ('Spam', 'Unsolicited commercial or repetitive content'),
    ('Violence', 'Content that promotes or glorifies violence'),
    ('Copyright Violation', 'Content that infringes on copyrighted material'),
    ('Personal Information', 'Sharing of private personal information without consent'),
    ('Other', 'Violation not covered by other categories');

CREATE TABLE reports (
    report_id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    reported_by_profile_id bigint NOT NULL,
    report_reason_id bigint NOT NULL,
    report_reason_extra text DEFAULT '' NOT NULL,
    created timestamp without time zone NOT NULL
);

ALTER TABLE reports ADD CONSTRAINT reports_report_reason_id_fkey FOREIGN KEY (report_reason_id) REFERENCES report_reasons(report_reason_id);
ALTER TABLE reports ADD CONSTRAINT reports_reported_by_profile_id_fkey FOREIGN KEY (reported_by_profile_id) REFERENCES profiles(profile_id);

CREATE TABLE actions (
	action_id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	title text NOT NULL UNIQUE,
	description text DEFAULT '' NOT NULL
);

INSERT INTO actions (title, description)
VALUES
    ('Warning', 'Issue a formal warning to the user without other penalties'),
    ('Hide Content', 'Hide the content from public view, but keep it in the database'),
    ('Delete Content', 'Permanently remove the flagged content from the database'),
    ('Temporary Ban', 'Temporarily suspend the user account for a defined period'),
    ('Permanent Ban', 'Permanently ban the user from the platform'),
    ('Shadow Ban', 'Make user posts only visible to themselves, not to other users'),
    ('Restrict Posting', 'Limit user''s ability to create new content for a period'),
    ('Require Approval', 'Require moderator approval before user content becomes visible'),
    ('Flag Account', 'Flag account for additional monitoring without direct action'),
    ('IP Ban', 'Ban all activity from the user''s IP address(es)'),
    ('No Action', 'Review complete - no action warranted');

CREATE TABLE moderator_actions (
	moderator_action_id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	action_id bigint NOT NULL,
	moderator_profile_id bigint NOT NULL,
	comment_id bigint,
	created timestamp without time zone NOT NULL,
	valid_from timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
	expires timestamp without time zone
);

ALTER TABLE moderator_actions ADD CONSTRAINT moderator_actions_action_id_fkey FOREIGN KEY (action_id) REFERENCES actions(action_id);
ALTER TABLE moderator_actions ADD CONSTRAINT moderator_actions_moderator_profile_id_fkey FOREIGN KEY (moderator_profile_id) REFERENCES profiles(profile_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE report_reasons;
DROP TABLE reports;
DROP TABLE actions;
DROP TABLE moderator_actions;

-- +goose StatementEnd
