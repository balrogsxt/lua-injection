# Lua-Injection

本项目是将`github.com/yuin/gopher-lua`库作为lua虚拟机,通过反射的方式将自己编写的结构体方法转换成对应的LuaTable函数进行快速调用

> 调用方法

```gopackage lua_injection

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"testing"
	"time"
)

// helper service
type HelperModule struct {
}

func (HelperModule) Name() string {
	return "helper"
}
func (HelperModule) GetA() *A {
	fmt.Println("call GetA()")
	return &A{}
}

// a service
type A struct {
}

func (A) Name() string {
	return "a"
}
func (A) GetB() *B {
	fmt.Println("call GetB()")
	return &B{}
}

// b service
type B struct {
	cb func(string)
}

func (b *B) Name() string {
	return "b"
}
func (b *B) Text() string {
	fmt.Println("call B Text")
	return time.Now().Format("2006-01-02 15:04:05")
}
func (b *B) Reg(cb func(string)) {
	b.cb = cb
}
func (b *B) Emit(v string) {
	if b.cb != nil {
		b.cb(v)
	} else {
		fmt.Println("未注册cb")
	}
}

func TestLua(t *testing.T) {

	state := lua.NewState()

	state.SetGlobal("helper", RefService(state, HelperModule{}))

	fmt.Println(state.DoString(`
		--调用测试
		print("text=",helper.getA().getB().text())

		--回调测试
		local b = helper.getA().getB()
		b.reg(function(val)
			print("emit",val)
		end)
		b.emit("触发值")
	`))

}

```