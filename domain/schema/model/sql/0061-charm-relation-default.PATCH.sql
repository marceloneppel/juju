-- Patch 0061: Add the is_default column to charm_relation and surface it on the
-- relation views.
--
-- Merge-forward note: when merging to the next major version, fold these
-- changes into the base DDL instead of keeping this PATCH file:
--   0015-charm.sql: add "is_default BOOLEAN DEFAULT FALSE" to charm_relation
--                   and cr.is_default to v_charm_relation.
--   0024-relation.sql: add cr.is_default to v_application_endpoint and
--                      v_relation_endpoint.

ALTER TABLE charm_relation ADD COLUMN is_default BOOLEAN DEFAULT FALSE;

DROP VIEW v_charm_relation;
CREATE VIEW v_charm_relation AS
SELECT
    cr.uuid,
    cr.charm_uuid,
    cr.name,
    crr.name AS role,
    cr.interface,
    cr.optional,
    cr.capacity,
    crs.name AS scope,
    cr.is_default
FROM charm_relation AS cr
JOIN charm_relation_role AS crr ON cr.role_id = crr.id
JOIN charm_relation_scope AS crs ON cr.scope_id = crs.id;

DROP VIEW v_application_endpoint;
CREATE VIEW v_application_endpoint AS
SELECT
    ae.uuid AS application_endpoint_uuid,
    cr.name AS endpoint_name,
    ae.application_uuid,
    a.name AS application_name,
    cr.interface,
    cr.optional,
    cr.capacity,
    crr.name AS role,
    crs.name AS scope,
    cr.is_default
FROM application_endpoint AS ae
JOIN application AS a ON ae.application_uuid = a.uuid
JOIN charm_relation AS cr ON ae.charm_relation_uuid = cr.uuid
JOIN charm_relation_role AS crr ON cr.role_id = crr.id
JOIN charm_relation_scope AS crs ON cr.scope_id = crs.id;

DROP VIEW v_relation_endpoint;
CREATE VIEW v_relation_endpoint AS
SELECT
    re.uuid AS relation_endpoint_uuid,
    re.endpoint_uuid AS application_endpoint_uuid,
    re.relation_uuid,
    ae.application_uuid,
    a.name AS application_name,
    cr.name AS endpoint_name,
    cr.interface,
    cr.optional,
    cr.capacity,
    crr.name AS role,
    crs.name AS scope,
    cr.is_default
FROM relation_endpoint AS re
JOIN relation AS r ON re.relation_uuid = r.uuid
JOIN application_endpoint AS ae ON re.endpoint_uuid = ae.uuid
JOIN application AS a ON ae.application_uuid = a.uuid
JOIN charm_relation AS cr ON ae.charm_relation_uuid = cr.uuid
JOIN charm_relation_role AS crr ON cr.role_id = crr.id
JOIN charm_relation_scope AS crs ON r.scope_id = crs.id;
