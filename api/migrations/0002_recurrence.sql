-- +goose Up
CREATE TYPE recur_unit AS ENUM ('day', 'week', 'month');

ALTER TABLE todos
    ADD COLUMN recur_unit     recur_unit,
    ADD COLUMN recur_interval integer,

    -- The schedule is counted from here, not from the last occurrence. A task
    -- anchored to 31 January lands on 28 February and then back on 31 March;
    -- the February row alone cannot say whether the anchor was the 28th or a
    -- clamped 31st, so every occurrence carries the anchor forward.
    ADD COLUMN recur_anchor   timestamptz,

    ADD CONSTRAINT recurrence_is_complete CHECK (
        (recur_unit IS NULL AND recur_interval IS NULL AND recur_anchor IS NULL)
        OR (recur_unit IS NOT NULL AND recur_interval >= 1)
    );

-- +goose Down
ALTER TABLE todos
    DROP CONSTRAINT recurrence_is_complete,
    DROP COLUMN recur_anchor,
    DROP COLUMN recur_interval,
    DROP COLUMN recur_unit;
DROP TYPE recur_unit;
