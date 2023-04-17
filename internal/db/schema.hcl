schema "main" {
}

table "users" {
  schema = schema.main
  column "id" {
    type = text
  }
  column "created_at" {
    type = datetime
  }
  column "updated_at" {
    type = datetime
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_users_created_at" {
    columns = [column.created_at]
  }
}

table "public_keys" {
  schema = schema.main
  column "id" {
    type = text
  }
  column "created_at" {
    type = datetime
  }
  column "updated_at" {
    type = datetime
  }
  column "fingerprint" {
    type = text
  }
  column "type" {
    type = text
  }
  column "user_id" {
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_public_keys_created_at" {
    columns = [column.created_at]
  }
  index "idx_pubkey_fingerprint" {
    unique  = true
    columns = [column.fingerprint]
  }
}

table "files" {
  schema = schema.main
  column "id" {
    type = text
  }
  column "created_at" {
    type = datetime
  }
  column "updated_at" {
    type = datetime
  }
  column "size" {
    type = integer
  }
  column "content" {
    type = blob
  }
  column "private" {
    type = numeric
  }
  column "type" {
    type = text
  }
  column "user_id" {
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_files_created_at" {
    columns = [column.created_at]
  }
}

