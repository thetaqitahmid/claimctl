DROP TRIGGER IF EXISTS trigger_log_reservation_change ON claimctl.reservations;
DROP TRIGGER IF EXISTS trigger_log_reservation_creation ON claimctl.reservations;

DROP FUNCTION IF EXISTS claimctl.log_reservation_change();

DROP INDEX IF EXISTS idx_reservation_history_resource;
DROP INDEX IF EXISTS idx_reservation_history_reservation;
DROP INDEX IF EXISTS idx_reservation_history_user;
DROP INDEX IF EXISTS idx_reservation_history_action;

DROP TABLE IF EXISTS claimctl.reservation_history;