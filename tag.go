package lua_injection

import (
	"reflect"
	"strings"
)

const LuaTag = "lua"

func getFieldKey(field reflect.StructField) string {
	key := field.Name
	if tag := field.Tag.Get(LuaTag); len(strings.Trim(tag, " ")) > 0 {
		key = tag
	}
	return key
}
