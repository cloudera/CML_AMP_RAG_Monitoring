package sbhttp

import "github.com/gorilla/schema"

var SchemaDecoder = schema.NewDecoder()

func init() {
	SchemaDecoder.SetAliasTag("json")
}
