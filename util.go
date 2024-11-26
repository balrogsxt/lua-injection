package lua_injection

import lua "github.com/yuin/gopher-lua"

// TableToInt64List Table转[]int64
func TableToInt64List(v lua.LValue) []int64 {
	list := make([]int64, 0)
	t, has := v.(*lua.LTable)
	if !has {
		return list
	}
	if t == nil {
		return list
	}
	t.ForEach(func(_ lua.LValue, val lua.LValue) {
		list = append(list, NewVar(val.String()).Int64())
	})
	return list
}

type TransformTable struct {
	table *lua.LTable
}

func LoadTransform(tab *lua.LTable) *TransformTable {
	return &TransformTable{
		table: tab,
	}
}
func (c *TransformTable) GetValue(field string) lua.LValue {
	if c.Get() == nil {
		return lua.LNil
	}
	return c.table.RawGetString(field)
}
func (c *TransformTable) Get() *lua.LTable {
	if c.table == nil || c.table == lua.LNil {
		return nil
	}
	return c.table
}
func (c *TransformTable) GetVar(field string, def ...any) *Var {
	if c.Get() == nil {
		var d any = nil
		if len(def) > 0 {
			d = def[0]
		}
		return NewVar(d)
	}
	v := c.Get().RawGetString(field)
	return NewVar(Unmarshal(v))
}
