package scripts

const V14AdminUserGrants = `BEGIN;

CREATE USER admin_api;

GRANT SELECT, INSERT, UPDATE ON DATABASE eventstore TO admin_api;
GRANT SELECT, INSERT, UPDATE ON TABLE eventstore.* TO admin_api;

COMMIT;`
