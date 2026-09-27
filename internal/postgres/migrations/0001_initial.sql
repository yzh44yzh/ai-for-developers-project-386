-- +goose Up
CREATE TABLE users (
    id          uuid PRIMARY KEY,
    first_name  text NOT NULL,
    last_name   text NOT NULL,
    description text NOT NULL DEFAULT '',
    email       text NOT NULL UNIQUE
);

CREATE TABLE meetings (
    id               uuid PRIMARY KEY,
    owner_id         uuid NOT NULL REFERENCES users (id),
    title            text NOT NULL,
    start            timestamptz NOT NULL,
    duration_minutes integer NOT NULL CHECK (duration_minutes > 0),
    description      text NOT NULL DEFAULT '',
    cancelled_at     timestamptz
);

CREATE TABLE meeting_guests (
    meeting_id uuid NOT NULL REFERENCES meetings (id),
    user_id    uuid NOT NULL REFERENCES users (id),
    PRIMARY KEY (meeting_id, user_id)
);

-- +goose Down
DROP TABLE meeting_guests, meetings, users;
