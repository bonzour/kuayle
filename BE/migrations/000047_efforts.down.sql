DO $$ BEGIN
 LOCK TABLE workspace_effort_settings,issue_efforts,effort_worklogs,effort_events,effort_operations IN ACCESS EXCLUSIVE MODE;
 IF EXISTS(SELECT 1 FROM workspace_effort_settings) OR EXISTS(SELECT 1 FROM issue_efforts)
    OR EXISTS(SELECT 1 FROM effort_worklogs) OR EXISTS(SELECT 1 FROM effort_events)
    OR EXISTS(SELECT 1 FROM effort_operations) THEN
  RAISE EXCEPTION 'Cannot remove Efforts while data or retry receipts exist';
 END IF;
END $$;
DROP TABLE effort_operations;
DROP TABLE effort_events;
DROP TABLE effort_worklogs;
DROP FUNCTION kuayle_effort_worklog_identity_guard();
DROP TABLE issue_efforts;
DROP TABLE workspace_effort_settings;
DROP FUNCTION kuayle_effort_timezone_guard();
