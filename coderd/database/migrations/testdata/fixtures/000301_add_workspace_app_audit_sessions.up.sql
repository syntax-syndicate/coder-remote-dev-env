INSERT INTO workspace_app_audit_sessions
	(agent_id, app_id, user_id, ip, user_agent, slug_or_port, status_code, started_at, updated_at)
VALUES
	('45e89705-e09d-4850-bcec-f9a937f5d78d', '36b65d0c-042b-4653-863a-655ee739861c', '30095c71-380b-457a-8995-97b8ee6e5307', '127.0.0.1', 'curl', '', 200, '2025-03-04 15:08:38.579772+02', '2025-03-04 15:06:48.755158+02'),
	('45e89705-e09d-4850-bcec-f9a937f5d78d', '36b65d0c-042b-4653-863a-655ee739861c', '00000000-0000-0000-0000-000000000000', '127.0.0.1', 'curl', '', 200, '2025-03-04 15:08:44.411389+02', '2025-03-04 15:08:44.411389+02'),
	('45e89705-e09d-4850-bcec-f9a937f5d78d', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', '::1', 'curl', 'terminal', 0, '2025-03-04 15:25:55.555306+02', '2025-03-04 15:25:55.555306+02');
