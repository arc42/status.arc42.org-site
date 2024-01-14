# configuration
# see https://atlasgo.io/atlas-schema/projects
#

# set the authorization token,
# see https://atlasgo.io/guides/sqlite/turso
variable "token" {
  type    = string
  default = getenv("TURSO_AUTH_TOKEN")
}

// Define an environment named "dev"
env "dev" {
  // Declare where the schema definition resides.
  // Also supported: ["file://multi.hcl", "file://schema.hcl"].
  src = "file://schema.hcl"

  // Define the URL of the database which is managed in this environment.
  url = "sqlite://dev.db?_fk=1"

  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/concepts/dev-database
  dev = "sqlite://file?mode=memory&_fk=1"
}

env "prod" {
  // Declare where the schema definition resides.
  src = "file://schema.hcl"

  // Define the URL of the database which is managed in this environment.
  url = "libsql+ws://arc42-statistics-gernotstarke.turso.io?authToken=${var.token}"

  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/concepts/dev-database
  dev = "sqlite://file?mode=memory&_fk=1"
}

