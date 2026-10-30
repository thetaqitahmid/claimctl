DROP TRIGGER reservation_webhook_intent ON claimctl.reservations;
DROP FUNCTION claimctl.enqueue_reservation_webhooks();
DROP TABLE claimctl.webhook_outbox;
DROP TABLE claimctl.worker_leases;
