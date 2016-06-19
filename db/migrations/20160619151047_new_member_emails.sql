-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

UPDATE update_types
   SET email_body_text = 'Hi {{.ForProfile.ProfileName}},

A new comment has been made on {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
'
 WHERE update_type_id = 1;

INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (9, 'new_user', 'When a new user is created (new sign-in from unrecognised email) and you are watching the People page', 'New user on {{.SiteTitle}}', 'Hi {{.ForProfile.ProfileName}},

A new user has signed-in on {{.SiteTitle}}

Their profile: {{.ProtoAndHost}}/profiles/{{.ByProfile.ID}}/

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new user has signed-in on <a href="{{.ProtoAndHost}}">{{.SiteTitle}}</a>.</p>

  <p>Their profile: <a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></p>
  
  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');

INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (9, true, false);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

UPDATE update_types
   SET email_body_text = 'Hi {{.ForProfile.ProfileName}},

A new comment has been made on {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/settings/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
'
 WHERE update_type_id = 1;

DELETE FROM update_options_defaults WHERE update_type_id = 9;
DELETE FROM update_types WHERE update_type_id = 9;
