CREATE TABLE rule (
    sid integer PRIMARY KEY NOT NULL,
    rev smallint,
    file text NOT NULL,
    pattern text NOT NULL,
    updated_at timestamp NOT NULL DEFAULT NOW(),
    active boolean NOT NULL
);

CREATE TABLE rule_comment (
    id serial PRIMARY KEY NOT NULL,
    sid integer,
    rev smallint,
    comment text NOT NULL
);

CREATE TABLE rule_archive (
    sid integer PRIMARY KEY NOT NULL,
    rev smallint,
    file text NOT NULL,
    pattern text NOT NULL,
    deleted_at timestamp NOT NULL DEFAULT NOW()
);
