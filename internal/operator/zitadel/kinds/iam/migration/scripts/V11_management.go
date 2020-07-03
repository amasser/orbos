package scripts

const V11Management = `BEGIN;

CREATE DATABASE management;

COMMIT;

BEGIN;

CREATE USER management;

GRANT SELECT, INSERT, UPDATE, DELETE ON DATABASE management TO management;
GRANT SELECT, INSERT, UPDATE ON DATABASE eventstore TO management;
GRANT SELECT, INSERT, UPDATE ON TABLE eventstore.* TO management;

COMMIT;
`
