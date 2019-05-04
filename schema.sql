CREATE TABLE rule (
    sid integer PRIMARY KEY NOT NULL,
    rev smallint,
    file text NOT NULL,
    pattern text NOT NULL,
    updated_at timestamp NOT NULL DEFAULT NOW(),
    deactivated_at timestamp NULL DEFAULT NULL,
    deleted_at timestamp NULL DEFAULT NULL
);

CREATE TABLE rule_comment (
    id serial PRIMARY KEY NOT NULL,
    sid integer REFERENCES rule(sid),
    rev smallint,
    commented_at timestamp NOT NULL DEFAULT NOW(),
    comment text NOT NULL
);
