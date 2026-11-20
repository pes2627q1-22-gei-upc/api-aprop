DROP TRIGGER IF EXISTS userratings_refresh_user_rating ON UserRatings;
DROP TRIGGER IF EXISTS eventratings_refresh_event_rating ON EventRatings;
DROP TRIGGER IF EXISTS eventratings_refresh_organizer_rating ON EventRatings;

DROP FUNCTION IF EXISTS refresh_organizer_rating();
DROP FUNCTION IF EXISTS refresh_event_rating();

ALTER TABLE Events DROP COLUMN IF EXISTS Rating;

CREATE OR REPLACE FUNCTION refresh_organizer_rating() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
BEGIN
    UPDATE Users u SET Rating = (
        SELECT AVG(er.Rating)::real
        FROM EventRatings er
        JOIN UnofficialEvents ue ON ue.ID = er.Event
        WHERE ue.Creator = u.ID
    )
    WHERE u.ID = (SELECT Creator FROM UnofficialEvents WHERE ID = COALESCE(NEW.Event, OLD.Event));
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS eventratings_refresh_organizer_rating ON EventRatings;
CREATE TRIGGER eventratings_refresh_organizer_rating
    AFTER INSERT OR UPDATE OR DELETE ON EventRatings
    FOR EACH ROW EXECUTE FUNCTION refresh_organizer_rating();

-- Backfill for the ratings that already exist.
UPDATE Users u SET Rating = (
    SELECT AVG(er.Rating)::real
    FROM EventRatings er
    JOIN UnofficialEvents ue ON ue.ID = er.Event
    WHERE ue.Creator = u.ID
);