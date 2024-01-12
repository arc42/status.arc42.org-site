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
  column "route"{
    null = false
    type = varchar(50)
  }
}

table "time_of_plausible_call" {
  schema = schema.main
  column "invocation_time"{
    null = false
    type = datetime
  }
}

table "time_of_github_call" {
  schema = schema.main
  column "invocation_time"{
    null = false
    type = datetime
  }
}