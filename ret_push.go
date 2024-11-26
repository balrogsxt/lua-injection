package lua_injection

import lua "github.com/yuin/gopher-lua"

func Ret(state *lua.LState, values ...any) int {
	lv := make([]lua.LValue, 0, len(values))
	for _, v := range values {
		lv = append(lv, Marshal(state, v))
	}
	return RetValue(state, lv...)
}
func RetValue(state *lua.LState, v ...lua.LValue) int {
	for _, r := range v {
		state.Push(r)
	}
	return len(v)
}
