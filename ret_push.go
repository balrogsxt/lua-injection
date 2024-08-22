package lua_injection

import lua "github.com/yuin/gopher-lua"

func ret(state *lua.LState, values ...any) int {
	lv := make([]lua.LValue, 0, len(values))
	for _, v := range values {
		lv = append(lv, Marshal(state, v))
	}
	return retValue(state, lv...)
}
func retValue(state *lua.LState, v ...lua.LValue) int {
	for _, r := range v {
		state.Push(r)
	}
	return len(v)
}
