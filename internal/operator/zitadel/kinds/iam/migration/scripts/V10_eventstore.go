package scripts

const V10Eventstore = `BEGIN;

CREATE DATABASE eventstore;

COMMIT;


BEGIN;

CREATE USER eventstore;

GRANT SELECT, INSERT, UPDATE ON DATABASE eventstore TO eventstore;

COMMIT;

BEGIN;

CREATE SEQUENCE eventstore.event_seq;

COMMIT;

BEGIN;

CREATE TABLE eventstore.events (
    id UUID DEFAULT gen_random_uuid(),
    
    event_type TEXT,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    aggregate_version TEXT NOT NULL,
    event_sequence BIGINT NOT NULL DEFAULT nextval('eventstore.event_seq'),
    previous_sequence BIGINT UNIQUE,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    event_data JSONB,
    editor_user TEXT NOT NULL, 
    editor_service TEXT NOT NULL,
    resource_owner TEXT NOT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE eventstore.locks (
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    until TIMESTAMPTZ,
    UNIQUE (aggregate_type, aggregate_id)
);

COMMIT;
`
