package lua_injection

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"reflect"
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

type T1 struct {
}

func (T1) Name() string {
	return "t1"
}

type T1Params struct {
	Json map[string]any `lua:"json"`
}

func (T1) Params(v1 Value, v2 Value, params T1Params) {
	fmt.Println("v1=", v1) //这里要求接受lua.LTable
	v1t, has := v1.(*lua.LTable)
	if has {
		fmt.Println("luatable", v1t)
	}
	fmt.Println("v2=", v2, reflect.TypeOf(v2).String()) //这里要求拿到的是lua传过来的lua.LString

	fmt.Println("params", params.Json) //这里要求接收的是常规map[string]any
	fmt.Println("----------")
	for k, v := range params.Json {
		fmt.Println(k, v)
	}
}

func TestLua1(t *testing.T) {

	state := lua.NewState()

	state.SetGlobal("t1", RefService(state, T1{}))

	fmt.Println(state.DoString(`
		--调用测试
		t1.params({
			a = 1,
			b = "str"
		},"test",{
			json = {
				key = {"1","2","3"}
			}
		})
	`))

}
