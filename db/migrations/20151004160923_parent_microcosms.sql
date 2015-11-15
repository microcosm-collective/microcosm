
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE microcosms ADD COLUMN parent_id bigint REFERENCES microcosms(microcosm_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_microcosms_flags()
  RETURNS trigger AS
$BODY$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 2
               AND item_id = OLD.microcosm_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.is_sticky <> OLD.is_sticky OR
               NEW.parent_id <> OLD.parent_id THEN

            -- Microcosms
            UPDATE flags
               SET item_is_deleted = NEW.is_deleted
                  ,item_is_moderated = NEW.is_moderated
                  ,item_is_sticky = NEW.is_sticky
                  ,last_modified = COALESCE(NEW.edited, NOW())
                  ,microcosm_id = NEW.parent_id 
             WHERE item_type_id = 2
               AND item_id = NEW.microcosm_id;

            -- Children (items)
            UPDATE flags
               SET microcosm_is_deleted = NEW.is_deleted
                  ,microcosm_is_moderated = NEW.is_moderated
             WHERE microcosm_id = NEW.microcosm_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,item_type_id
               ,item_id
               ,item_is_deleted
               ,item_is_moderated
               ,item_is_sticky
               ,last_modified
               ,created_by
               ,microcosm_id
            )
            SELECT NEW.site_id
                  ,2
                  ,NEW.microcosm_id
                  ,NEW.is_deleted
                  ,NEW.is_moderated
                  ,NEW.is_sticky
                  ,NEW.created
                  ,NEW.created_by
                  ,NEW.parent_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_microcosms_search_index()
  RETURNS trigger AS
$BODY$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 2
               AND item_id = OLD.microcosm_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN
            IF NEW.title <> OLD.title OR
               NEW.description <> OLD.description OR
               NEW.parent_id <> OLD.parent_id THEN
        
                UPDATE search_index
                   SET title_text = NEW.title
                      ,title_vector = setweight(to_tsvector(NEW.title), 'A')
                      ,document_text = NEW.title || ' ' || NEW.description
                      ,document_vector = setweight(to_tsvector(NEW.title), 'A') || setweight(to_tsvector(NEW.description), 'D')
                      ,last_modified = NOW()
                      ,microcosm_id = NEW.parent_id
                 WHERE item_type_id = 2
                   AND item_id = NEW.microcosm_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,profile_id
               ,item_type_id
               ,item_id
               ,title_text

               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
               ,microcosm_id
            ) VALUES (
                NEW.site_id
               ,NEW.created_by
               ,2
               ,NEW.microcosm_id
               ,NEW.title

               ,setweight(to_tsvector(NEW.title), 'A')
               ,NEW.title || ' ' || NEW.description
               ,setweight(to_tsvector(NEW.title), 'A') || setweight(to_tsvector(NEW.description), 'D')
               ,NEW.created
               ,NEW.parent_id
            );

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
-- +goose StatementEnd

DROP TRIGGER microcosms_flags ON microcosms;

CREATE TRIGGER microcosms_flags
  AFTER INSERT OR UPDATE OR DELETE
  ON microcosms
  FOR EACH ROW
  EXECUTE PROCEDURE update_microcosms_flags();

DROP TRIGGER microcosms_search_index ON microcosms;

CREATE TRIGGER microcosms_search_index
  AFTER INSERT OR UPDATE OR DELETE
  ON microcosms
  FOR EACH ROW
  EXECUTE PROCEDURE update_microcosms_search_index();

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_microcosms_flags()
  RETURNS trigger AS
$BODY$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 2
               AND item_id = OLD.microcosm_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.is_sticky <> OLD.is_sticky THEN

            -- Microcosms
            UPDATE flags
               SET item_is_deleted = NEW.is_deleted
                  ,item_is_moderated = NEW.is_moderated
                  ,item_is_sticky = NEW.is_sticky
                  ,last_modified = COALESCE(NEW.edited, NOW())
             WHERE item_type_id = 2
               AND item_id = NEW.microcosm_id;

            -- Children (items)
            UPDATE flags
               SET microcosm_is_deleted = NEW.is_deleted
                  ,microcosm_is_moderated = NEW.is_moderated
             WHERE microcosm_id = NEW.microcosm_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,item_type_id
               ,item_id
               ,item_is_deleted
               ,item_is_moderated
               ,item_is_sticky
               ,last_modified
               ,created_by
            )
            SELECT NEW.site_id
                  ,2
                  ,NEW.microcosm_id
                  ,NEW.is_deleted
                  ,NEW.is_moderated
                  ,NEW.is_sticky
                  ,NEW.created
                  ,NEW.created_by;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_microcosms_search_index()
  RETURNS trigger AS
$BODY$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 2
               AND item_id = OLD.microcosm_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN
            IF NEW.title <> OLD.title OR NEW.description <> OLD.description THEN
        
                UPDATE search_index
                   SET title_text = NEW.title
                      ,title_vector = setweight(to_tsvector(NEW.title), 'A')
                      ,document_text = NEW.title || ' ' || NEW.description
                      ,document_vector = setweight(to_tsvector(NEW.title), 'A') || setweight(to_tsvector(NEW.description), 'D')
                      ,last_modified = NOW()
                 WHERE item_type_id = 2
                   AND item_id = NEW.microcosm_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,profile_id
               ,item_type_id
               ,item_id
               ,title_text

               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
            ) VALUES (
                NEW.site_id
               ,NEW.created_by
               ,2
               ,NEW.microcosm_id
               ,NEW.title

               ,setweight(to_tsvector(NEW.title), 'A')
               ,NEW.title || ' ' || NEW.description
               ,setweight(to_tsvector(NEW.title), 'A') || setweight(to_tsvector(NEW.description), 'D')
               ,NEW.created
            );

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
-- +goose StatementEnd

DROP TRIGGER microcosms_flags ON microcosms;

CREATE TRIGGER microcosms_flags
  AFTER INSERT OR UPDATE OR DELETE
  ON microcosms
  FOR EACH ROW
  EXECUTE PROCEDURE update_microcosms_flags();

DROP TRIGGER microcosms_search_index ON microcosms;

CREATE TRIGGER microcosms_search_index
  AFTER INSERT OR UPDATE OR DELETE
  ON microcosms
  FOR EACH ROW
  EXECUTE PROCEDURE update_microcosms_search_index();

ALTER TABLE microcosms DROP CONSTRAINT microcosms_parent_id_fkey;
ALTER TABLE microcosms DROP COLUMN parent_id;
