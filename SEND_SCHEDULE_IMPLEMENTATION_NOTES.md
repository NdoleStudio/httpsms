# Send Schedule implementation notes

What was added:
- Backend entities, repository, service, validator, and handler for `send-schedules`
- Message send-time resolution against the user's default active schedule
- Basic outstanding-message gating by `scheduled_send_time`
- Settings UI section for listing, creating, editing, deleting, and setting the default schedule

New backend routes:
- `GET /v1/send-schedules`
- `POST /v1/send-schedules`
- `GET /v1/send-schedules/:scheduleID`
- `PUT /v1/send-schedules/:scheduleID`
- `DELETE /v1/send-schedules/:scheduleID`
- `POST /v1/send-schedules/:scheduleID/default`

