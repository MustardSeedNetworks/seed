-- 00026_user_dashboards.sql — each user's dashboard layout (UI-SEED-22).
--
-- widgets is a JSON array of widget ids in display order. The widget catalog
-- belongs to the UI, so the store keeps ids rather than a foreign key, and the
-- row goes with its user. No row means the user never saved a layout and sees
-- the default one; an empty array is a deliberately empty dashboard.

-- +goose Up
CREATE TABLE user_dashboards (
	user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	widgets TEXT NOT NULL CHECK (json_valid(widgets) AND json_type(widgets) = 'array'),
	updated_at TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE user_dashboards;
