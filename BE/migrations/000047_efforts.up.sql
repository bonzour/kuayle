-- Settings are lazy: a missing row means enabled / Europe/Rome / revision 0.
CREATE TABLE workspace_effort_settings (
 workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 enabled BOOLEAN NOT NULL DEFAULT TRUE,
 timezone TEXT NOT NULL DEFAULT 'Europe/Rome',
 revision BIGINT NOT NULL DEFAULT 0 CHECK (revision BETWEEN 0 AND 9007199254740991),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE FUNCTION kuayle_effort_timezone_guard() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_timezone_names WHERE name=NEW.timezone)
    OR (NEW.timezone<>'UTC' AND position('/' IN NEW.timezone)=0)
    OR NEW.timezone LIKE 'posix/%' OR NEW.timezone LIKE 'right/%' THEN
  RAISE EXCEPTION 'Efforts requires an IANA timezone' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER effort_timezone BEFORE INSERT OR UPDATE OF timezone ON workspace_effort_settings
 FOR EACH ROW EXECUTE FUNCTION kuayle_effort_timezone_guard();

CREATE TABLE issue_efforts (
 issue_id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL,
 estimate_minutes INTEGER CHECK (estimate_minutes>=0),
 remaining_minutes INTEGER CHECK (remaining_minutes>=0),
 revision BIGINT NOT NULL DEFAULT 0 CHECK (revision BETWEEN 0 AND 9007199254740991),
 FOREIGN KEY (workspace_id,issue_id) REFERENCES issues(workspace_id,id) ON DELETE CASCADE
);

-- Historical UUIDs deliberately do not reference users or workspace_members:
-- removing an account or membership must not erase or reassign its work.
CREATE TABLE effort_worklogs (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL,
 issue_id UUID NOT NULL,
 owner_user_id UUID NOT NULL CHECK (owner_user_id<>'00000000-0000-0000-0000-000000000000'),
 created_by_user_id UUID NOT NULL CHECK (created_by_user_id<>'00000000-0000-0000-0000-000000000000'),
 work_date DATE NOT NULL CHECK (work_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
 duration_minutes INTEGER NOT NULL CHECK (duration_minutes BETWEEN 1 AND 1440),
 description TEXT CHECK (char_length(description)<=2000),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 FOREIGN KEY (workspace_id,issue_id) REFERENCES issues(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX effort_worklogs_issue_page ON effort_worklogs(workspace_id,issue_id,work_date DESC,created_at DESC,id DESC);
CREATE INDEX effort_worklogs_daily_budget ON effort_worklogs(workspace_id,owner_user_id,work_date);
CREATE FUNCTION kuayle_effort_worklog_identity_guard() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.id<>OLD.id OR NEW.workspace_id<>OLD.workspace_id OR NEW.issue_id<>OLD.issue_id
    OR NEW.owner_user_id<>OLD.owner_user_id OR NEW.created_by_user_id<>OLD.created_by_user_id
    OR NEW.created_at<>OLD.created_at THEN
  RAISE EXCEPTION 'Worklog identity and initial attribution are immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER effort_worklog_identity BEFORE UPDATE ON effort_worklogs
 FOR EACH ROW EXECUTE FUNCTION kuayle_effort_worklog_identity_guard();

CREATE TABLE effort_events (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL,
 issue_id UUID NOT NULL,
 event_type TEXT NOT NULL CHECK (event_type IN ('estimate_set','remaining_set','worklog_created','worklog_updated','worklog_deleted')),
 actor_kind TEXT NOT NULL CHECK (actor_kind IN ('user','system')),
 actor_user_id UUID,
 system_source TEXT,
 cause TEXT NOT NULL CHECK (cause IN ('manual','worklog','closure','reactivation')),
 snapshot_version INTEGER NOT NULL DEFAULT 1 CHECK (snapshot_version=1),
 before_snapshot JSONB CHECK (jsonb_typeof(before_snapshot)='object'),
 after_snapshot JSONB CHECK (jsonb_typeof(after_snapshot)='object'),
 revision BIGINT NOT NULL CHECK (revision BETWEEN 1 AND 9007199254740991),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 CHECK (before_snapshot IS NOT NULL OR after_snapshot IS NOT NULL),
 CHECK ((actor_kind='user' AND actor_user_id IS NOT NULL
         AND actor_user_id<>'00000000-0000-0000-0000-000000000000' AND system_source IS NULL)
     OR (actor_kind='system' AND actor_user_id IS NULL AND length(btrim(system_source))>0 AND system_source IS NOT NULL)),
 FOREIGN KEY (workspace_id,issue_id) REFERENCES issues(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX effort_events_issue ON effort_events(workspace_id,issue_id,created_at DESC,id DESC);

-- Only a fingerprint and receipt survive issue deletion; never store request
-- bodies, descriptions or history here. There is intentionally no issue FK.
CREATE TABLE effort_operations (
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_user_id UUID NOT NULL CHECK (actor_user_id<>'00000000-0000-0000-0000-000000000000'),
 request_id UUID NOT NULL CHECK (request_id<>'00000000-0000-0000-0000-000000000000'),
 request_hash BYTEA NOT NULL CHECK (octet_length(request_hash)=32),
 issue_id UUID,
 resource_id UUID NOT NULL,
 revision BIGINT NOT NULL CHECK (revision BETWEEN 1 AND 9007199254740991),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (workspace_id,actor_user_id,request_id)
);

-- Migration 46's installation loop cannot cover tables created afterwards.
DO $$ DECLARE tab TEXT; BEGIN
 FOREACH tab IN ARRAY ARRAY['workspace_effort_settings','issue_efforts','effort_worklogs','effort_events','effort_operations'] LOOP
  EXECUTE format('CREATE TRIGGER privacy_statement_shared BEFORE INSERT OR UPDATE OR DELETE ON %I FOR EACH STATEMENT EXECUTE FUNCTION kuayle_privacy_statement_guard(''shared'')',tab);
 END LOOP;
END $$;
