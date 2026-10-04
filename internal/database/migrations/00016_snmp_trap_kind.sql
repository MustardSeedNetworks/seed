-- 00016_snmp_trap_kind.sql — the trap listener's event kind is "snmp-trap"
-- (P-B4, #1376).
--
-- It was "snmp-trap-v2c" while the listener took v1 and v2c only. Once it
-- also accepts SNMPv3 the old name misdescribes every v3 event, so stored
-- events and the rules that match on the kind move to the new name with it.

-- +goose Up
UPDATE listener_events SET kind = 'snmp-trap' WHERE kind = 'snmp-trap-v2c';
UPDATE alert_rules SET match_kind = 'snmp-trap' WHERE match_kind = 'snmp-trap-v2c';

-- +goose Down
UPDATE alert_rules SET match_kind = 'snmp-trap-v2c' WHERE match_kind = 'snmp-trap';
UPDATE listener_events SET kind = 'snmp-trap-v2c' WHERE kind = 'snmp-trap';
