-- Users.Rating = mean of the EventRatings received by the user's NoOficial events (NULL = none).
-- The trigger keeps it up to date on every rating insert, edit and delete.
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
