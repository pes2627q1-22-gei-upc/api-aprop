ALTER TABLE events add column if not exists rating real;

DROP TRIGGER IF EXISTS eventratings_refresh_organizer_rating ON EventRatings;

CREATE OR REPLACE FUNCTION refresh_organizer_rating() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    user_id uuid;
BEGIN
    user_id := CASE
        WHEN TG_OP = 'DELETE' THEN OLD.Rated
        ELSE NEW.Rated
    END;
    UPDATE Users u SET Rating = (
        SELECT AVG(rating)::real
        FROM UserRatings
        WHERE rated = u.ID
    )
    WHERE u.ID = user_id;
    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION refresh_event_rating() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    event_id integer;
BEGIN
    event_id := CASE
        WHEN TG_OP = 'DELETE' THEN OLD.Event
        ELSE NEW.Event
    END;
    UPDATE Events e SET Rating = (
        SELECT AVG(rating)::real
        FROM EventRatings
        WHERE event = e.ID
    )
    WHERE e.ID = event_id;
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS userratings_refresh_user_rating ON UserRatings;
CREATE TRIGGER userratings_refresh_user_rating
    AFTER INSERT OR UPDATE OR DELETE ON UserRatings
    FOR EACH ROW EXECUTE FUNCTION refresh_organizer_rating();

DROP TRIGGER IF EXISTS eventratings_refresh_event_rating ON EventRatings;
CREATE TRIGGER eventratings_refresh_event_rating
    AFTER INSERT OR UPDATE OR DELETE ON EventRatings
    FOR EACH ROW EXECUTE FUNCTION refresh_event_rating();

UPDATE Users u SET Rating = (
    SELECT AVG(ur.Rating)::real
    FROM UserRatings ur
    WHERE ur.Rated = u.ID
);

UPDATE Events e SET Rating = (
    SELECT AVG(er.Rating)::real
    FROM EventRatings er
    WHERE er.Event = e.ID
);