#
schema "main" {
}


table "system_startup" {
  schema = schema.main
  column "startup" {
    null = false
    type = datetime
  }
  column "app_version" {
    null = false
    type = varchar(15)
  }
  column "environment" {
    null = false
    type = varchar(15)
  }
}

#
table "time_of_invocation" {
  schema = schema.main
  column "invocation_time" {
    null = false
    type = datetime
  }
  column "request_ip" {
    null = false
    type = varchar(16)
  }
  column "route" {
    null = false
    type = varchar(50)
  }
}

table "time_of_plausible_call" {
  schema = schema.main
  column "invocation_time" {
    null = false
    type = datetime
  }
}

table "time_of_github_call" {
  schema = schema.main
  column "invocation_time" {
    null = false
    type = datetime
  }
}

# status history events

table "status_snapshot" {
  schema = schema.main
  column "id" {
    null = false
    type = integer
  }
  column "site" {
    null = false
    type = varchar(100)
  }
  column "monitor_id" {
    null = true
    type = varchar(64)
  }
  column "status" {
    null = false
    type = varchar(12)
  }
  column "changed_at" {
    null = false
    type = datetime
  }
  column "response_ms" {
    null = true
    type = integer
  }
  column "detail" {
    null = true
    type = varchar(255)
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_snapshot_site_changed" {
    columns = [column.site, column.changed_at]
  }
}

# aggregated buckets (optional)

table "status_bucket" {
  schema = schema.main
  column "site" {
    null = false
    type = varchar(100)
  }
  column "bucket_start" {
    null = false
    type = datetime
  }
  column "bucket_kind" {
    null = false
    type = varchar(8)
  }
  column "downtime_minutes" {
    null = false
    type = integer
    default = 0
  }
  column "outage_count" {
    null = false
    type = integer
    default = 0
  }
  primary_key {
    columns = [column.site, column.bucket_kind, column.bucket_start]
  }
}

# probe heartbeat: one row per prober run. Freshness on every surface is
# derived from the newest row here; if the workflow stops, this stops
# advancing and the page says "stale" instead of showing old green
# (ADR-0019, deviation noted there: heartbeat table instead of bucket
# timestamps).

table "probe_run" {
  schema = schema.main
  column "run_at" {
    null = false
    type = datetime
  }
  column "vantage" {
    null = false
    type = varchar(64)
  }
  column "sites_checked" {
    null = false
    type = integer
  }
  column "duration_ms" {
    null = false
    type = integer
  }
}
