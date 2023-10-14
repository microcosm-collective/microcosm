
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

INSERT INTO attendee_state (state_id, title) VALUES (1, 'yes');
INSERT INTO attendee_state (state_id, title) VALUES (2, 'maybe');
INSERT INTO attendee_state (state_id, title) VALUES (3, 'invited');
INSERT INTO attendee_state (state_id, title) VALUES (4, 'no');

INSERT INTO item_types (item_type_id, title, rank) VALUES (1, 'site', 10);
INSERT INTO item_types (item_type_id, title, rank) VALUES (2, 'microcosm', 8);
INSERT INTO item_types (item_type_id, title, rank) VALUES (3, 'profile', 9);
INSERT INTO item_types (item_type_id, title, rank) VALUES (4, 'comment', 2);
INSERT INTO item_types (item_type_id, title, rank) VALUES (5, 'huddle', 5);
INSERT INTO item_types (item_type_id, title, rank) VALUES (6, 'conversation', 4);
INSERT INTO item_types (item_type_id, title, rank) VALUES (7, 'poll', 5);
INSERT INTO item_types (item_type_id, title, rank) VALUES (8, 'article', 5);
INSERT INTO item_types (item_type_id, title, rank) VALUES (9, 'event', 6);
INSERT INTO item_types (item_type_id, title, rank) VALUES (10, 'q_and_a', 5);
INSERT INTO item_types (item_type_id, title, rank) VALUES (11, 'classified', 5);
INSERT INTO item_types (item_type_id, title, rank) VALUES (12, 'album', 5);
INSERT INTO item_types (item_type_id, title, rank) VALUES (13, 'attendee', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (14, 'user', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (15, 'attribute', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (16, 'update', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (17, 'role', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (18, 'update_type', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (19, 'watcher', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (20, 'auth', 0);
INSERT INTO item_types (item_type_id, title, rank) VALUES (21, 'attachment', 0);

INSERT INTO platform_options (send_email, send_sms) VALUES (true, false);

INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (3, '[\w]+\.strava\.[\w]+');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (4, '[\w]+\.garmin\.[\w]+');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (5, '(?:[\w]+\.)?vimeo\.[\w]+');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (6, 'www\.google\.com');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (1, '(?:[\w]+\.)?youtube\.[\w]+');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (2, '(?:[\w]+\.)?youtu\.be');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (7, '(?:[\w]+\.)?gpsies\.com');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (8, '(?:[\w]+\.)?everytrail\.com');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (9, '(?:[\w]+\.)?bikely\.com');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (10, '(?:[\w]+\.)?bikemap\.net');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (11, '(?:[\w]+\.)?ridewithgps\.com');
INSERT INTO rewrite_domains (domain_id, domain_regex) VALUES (12, '(?:[\w]+\.)?plotaroute\.com');

INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (1, 'YouTube', '^(?:https?:|)(?:\/\/)(?:www.|)(?:youtu\.be\/|youtube\.com(?:\/embed\/|\/v\/|\/watch\?v=|\/ytscreeningroom\?v=|\/feeds\/api\/videos\/|\/user\S*[^\w\-\s]))([\w\-]{11})(?:\W.*)?$', '<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/$1" frameborder="0" allowfullscreen></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (2, 'Strava Club Summary', 'https?://((app|www).strava.com/clubs/[a-z\-0-9]+/latest-rides/[a-f0-9]+(\?show_rides=false)?)', '<iframe height="160" width="460" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (3, 'Strava Recent Rides', 'https?://((app|www).strava.com/(clubs/[a-z\-0-9]+/latest-rides/[a-f0-9]+(\?show_rides=true)|athletes/[0-9]+/latest-rides/[a-f0-9]+))', '<iframe height="454" width="300" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>', true, 50);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (4, 'Strava Course Segment', 'https?://(?:app|www).strava.com/segments/[a-z0-9-]+-([0-9]+)', '<iframe height="360" width="460" frameborder="0" allowtransparency="true" scrolling="no" src="https://www.strava.com/segments/$1/embed"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (5, 'Strava Course Segment', 'https?://((?:app|www).strava.com/activities/[0-9]+/embed/[a-f0-9-]+)', '<iframe height="404" width="590" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (6, 'Strava Activity Summary', 'https?://((?:app|www).strava.com/athletes/[0-9]+/activity-summary/[a-f0-9-]+)', '<iframe height="360" width="360" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (7, 'Garmin Course', 'https?:\/\/connect\.garmin\.com\/course\/([0-9]+)', '<iframe width="560" height="600" frameborder="0" src="https://connect.garmin.com/course/embed/$1"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (8, 'Vimeo', 'https?://(?:[\w]+.)?vimeo.com\/(?:(?:groups\/[^\/]+\/videos\/)|(?:video\/))?([0-9]+).*', '<iframe src="https://player.vimeo.com/video/$1" width="500" height="281" frameborder="0" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (9, 'Garmin Activity', 'https?:\/\/connect\.garmin\.com\/(?:modern\/)activity\/([0-9]+)', '<iframe width="465" height="548" frameborder="0" src="https://connect.garmin.com/activity/embed/$1"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (10, 'Google Calendar', 'https?://(www.google.com/calendar/embed.*)', '<iframe src="https://$1" style="border: 0" width="100%" height="600" frameborder="0" scrolling="no"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (11, 'GPSies', 'http://www.gpsies.com/map.do\?fileId=([\w\d_-]+)', '<iframe src="http://www.gpsies.com/mapOnly.do?fileId=$1" width="600" height="400" frameborder="0" scrolling="no" marginheight="0" marginwidth="0"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (12, 'Every Trail', 'http://www.everytrail.com/view_trip.php\?trip_id=(\d+)', '<iframe src="https://www.everytrail.com/iframe2.php?trip_id=$1&width=600&height=400" marginheight="0" marginwidth="0" frameborder="0" scrolling="no" width="600" height="400"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (13, 'Bikely', 'http://www.bikely.com/maps/bike-path/([\w\d_-]+)', '<div id="routemapiframe" style="width: 100%; border: 1px solid #d0d0d0; background: #755; overflow: hidden; white-space: nowrap;"><iframe id="rmiframe" style="height:360px;  background: #eee;" width="100%" frameborder="0" scrolling="no" src="http://www.bikely.com/maps/bike-path/$1/embed/1"></iframe></div>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (14, 'Bikemap', 'http://www.bikemap.net/(?:en/)route/(\d+-?[a-zA-Z]*)/?', '<iframe src="http://www.bikemap.net/en/route/$1/widget/?width=640&amp;extended=1&amp;distance_markers=1&amp;height=480&amp;unit=metric" width="640" height="628" border="0" frameborder="0" marginheight="0" marginwidth="0" scrolling="no"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (15, 'Ride With GPS', 'http://ridewithgps.com/routes/([0-9]+)', '<iframe src="http://ridewithgps.com/routes/$1/embed" height="650px" width="100%" frameborder="0"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (16, 'Ride With GPS', 'http://ridewithgps.com/trips/([0-9]+)', '<iframe src="http://ridewithgps.com/trips/$1/embed" height="500px" width="100%" frameborder="0"></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (17, 'YouTube Playlists', '(?:http|https):\/\/(?:www.|)(?:youtu\.be|youtube\.com)\/.*[&?]list=([a-zA-Z0-9_]+)', '<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/videoseries?list=${1}" frameborder="0" allowfullscreen></iframe>', true, 50);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (18, 'Plot a Route', 'https?://www.plotaroute.com/route/([0-9]+)', '<iframe name="plotaroute_map_$1" src="https://www.plotaroute.com/embedmap/$1?units=km" frameborder="0" scrolling="no" width="100%" height="650px" allowfullscreen webkitallowfullscreen mozallowfullscreen oallowfullscreen msallowfullscreen></iframe>', true, 99);
INSERT INTO rewrite_rules (rule_id, name, match_regex, replace_regex, is_enabled, sequence) VALUES (19, 'Plot a Route Map', 'https?://www.plotaroute.com/map/([0-9]+)', '<iframe name="plotaroute_map_$1" src="https://www.plotaroute.com/embedmap/$1?units=km" frameborder="0" scrolling="no" width="100%" height="650px" allowfullscreen webkitallowfullscreen mozallowfullscreen oallowfullscreen msallowfullscreen></iframe>', true, 99);

SELECT pg_catalog.setval('url_rewrites_url_rewrite_id_seq', 19, true);

INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (1, 1);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (1, 17);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (2, 1);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (2, 17);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (3, 2);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (3, 3);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (3, 4);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (3, 5);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (3, 6);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (4, 7);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (4, 9);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (5, 8);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (6, 10);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (7, 11);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (8, 12);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (9, 13);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (10, 14);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (11, 15);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (11, 16);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (12, 18);
INSERT INTO rewrite_domain_rules (domain_id, rule_id) VALUES (12, 19);

SELECT pg_catalog.setval('rewrite_domains_domain_id_seq', 12, true);

INSERT INTO themes (theme_id, title, logo_url, background_url, background_color, background_position, link_color, favicon_url) VALUES (1, 'Microcosm Default', 'https://meta.microcosm.app/static/themes/1/logo.png', 'https://meta.microcosm.app/static/themes/1/background.png', '#FFFFFF', 'cover', '#4082C3', '/static/img/favico.png');

SELECT pg_catalog.setval('themes_theme_id_seq', 1, true);

INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (1, 'new_comment', 'When a comment has been posted in an item you are watching', 'New update on {{.SiteTitle}}', 'Hi {{.ForProfile.ProfileName}},

A new comment has been made on {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/settings/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new comment has been made on <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (2, 'reply_to_comment', 'When a comment of yours is replied to', 'New comment on {{.SiteTitle}} within {{.ContextText}}', 'Hi {{.ForProfile.ProfileName}},

A new reply has been made to your comment on {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new reply has been made to your comment on <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (3, 'mentioned', 'When you are @mentioned in a comment', '{{.ByProfile.ProfileName}} has mentioned you on {{.SiteTitle}}', 'Hi {{.ForProfile.ProfileName}},

You have been mentioned at {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>You have been mentioned at <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (4, 'new_comment_in_huddle', 'When you receive a new comment in a private message', 'New comment in a private message on {{.SiteTitle}}', 'Hi {{.ForProfile.ProfileName}},

A new comment has been added to a private message that you are a participant in: {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new comment has been added to a private message that you are a participant in <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (5, 'new_attendee', 'When an attendee added to an event you are watching', 'New attendee on {{.SiteTitle}} for the event {{.ContextText}}', 'Hi {{.ForProfile.ProfileName}},

A new attendee is coming to the event {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new attendee is coming to the event <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (6, 'new_vote', 'When a vote is cast in a poll you are watching', 'A new vote has been registered on {{.SiteTitle}} in the poll {{.ContextText}}', 'Hi {{.ForProfile.ProfileName}},

A new vote has been registered on the poll {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new vote has been registered on the poll <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (7, 'event_reminder', 'When an event you are attending is imminent', 'An event is imminent on {{.SiteTitle}}, {{.ContextText}}', 'Hi {{.ForProfile.ProfileName}},

An event is imminent, {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>An event is imminent <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');
INSERT INTO update_types (update_type_id, title, description, email_subject, email_body_text, email_body_html) VALUES (8, 'new_item', 'When a new item is created in a microcosm you are watching', 'A new item has been created on {{.SiteTitle}} in {{.ContextText}}', 'Hi {{.ForProfile.ProfileName}},

A new item has been created in {{.ContextText}} ({{.ContextLink}})

View other updates: {{.ProtoAndHost}}/updates/

Thanks,

{{.SiteTitle}}

Change your email settings: {{.ProtoAndHost}}/updates/settings/
', '<p>Hello {{.ForProfile.ProfileName}},<p>

  <p>A new item has been created in <a href="{{.ContextLink}}">{{.ContextText}}</a>.</p>

  <blockquote>
    {{.Body}}
    &mdash; <cite style="font-style:normal;"><a href="/profiles/{{.ByProfile.ID}}/">@{{.ByProfile.ProfileName}}</a></cite>
  </blockquote>

  <p><a href="{{.ProtoAndHost}}/updates/">View all of your updates</a>.</p>

  <p>Thanks,</p>

  <p>{{.SiteTitle}}</p>

  <p>Change <a href="{{.ProtoAndHost}}/updates/settings/">your email settings</a>.</p>');

INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (1, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (2, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (3, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (4, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (5, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (6, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (7, false, false);
INSERT INTO update_options_defaults (update_type_id, send_email, send_sms) VALUES (8, false, false);

INSERT INTO value_types (value_type_id, title) VALUES (1, 'string');
INSERT INTO value_types (value_type_id, title) VALUES (2, 'date');
INSERT INTO value_types (value_type_id, title) VALUES (3, 'number');
INSERT INTO value_types (value_type_id, title) VALUES (4, 'boolean');

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

TRUNCATE attendee_state CASCADE;
TRUNCATE item_types CASCADE;
TRUNCATE platform_options CASCADE;
TRUNCATE rewrite_domains CASCADE;
TRUNCATE rewrite_rules CASCADE;
TRUNCATE rewrite_domain_rules CASCADE;
TRUNCATE themes CASCADE;
TRUNCATE update_types CASCADE;
TRUNCATE update_options_defaults CASCADE;
TRUNCATE value_types CASCADE;

ALTER SEQUENCE rewrite_domains_domain_id_seq RESTART WITH 1;
ALTER SEQUENCE themes_theme_id_seq RESTART WITH 1;
ALTER SEQUENCE url_rewrites_url_rewrite_id_seq RESTART WITH 1;
