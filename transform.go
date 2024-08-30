package lua_injection

import (
	"context"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"log"
	"reflect"
	"slices"
)

// Marshal 将go变量转换成Lua变量
func Marshal(state *lua.LState, v any) lua.LValue {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Marshal Lua Error: ", err)
		}
	}()
	if v == context.DeadlineExceeded {
		return lua.LString(context.DeadlineExceeded.Error())
	}
	tp := reflect.TypeOf(v)
	vp := reflect.ValueOf(v)

	switch vp.Kind() {
	case reflect.Slice: //切片
		table := state.NewTable()
		if vp.IsValid() {
			for i := 0; i < vp.Len(); i++ {
				item := vp.Index(i).Interface()
				table.RawSetInt(i+1, Marshal(state, item))
			}
		}
		return table
	case reflect.Struct: //结构体
		table := state.NewTable()
		//结构体字段
		for i := 0; i < tp.NumField(); i++ {
			field := tp.Field(i)
			value := vp.Field(i)
			if !field.IsExported() { //非公开的不要
				continue
			}
			key := getFieldKey(field)
			table.RawSetString(key, Marshal(state, value.Interface()))
		}
		return table
	case reflect.Map:
		table := state.NewTable()
		iter := vp.MapRange()
		for iter.Next() {
			key := iter.Key().Interface()
			value := iter.Value().Interface()
			//判断key的类型是不是数字否则字符串
			switch reflect.TypeOf(key).Kind() {
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
				table.RawSetInt(NewVar(key).Int(), Marshal(state, value))
				break
			default:
				table.RawSetString(NewVar(key).String(), Marshal(state, value))
				break
			}
		}
		return table
	case reflect.String:
		return lua.LString(NewVar(vp.Interface()).String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(NewVar(vp.Interface()).Int64())
	case reflect.Float32, reflect.Float64: //浮点均为字符串处理
		return lua.LString(NewVar(vp.Interface()).String())
	case reflect.Bool:
		return lua.LBool(NewVar(vp.Interface()).Bool())
	case reflect.Ptr:
		if e, h := v.(error); h {
			return lua.LString(e.Error())
		}
		if t, h := v.(*lua.LTable); h {
			return t
		}
		if t, has := v.(IService); has { //指针类型才进行方法解析
			//判断IService是否是nil
			if reflect.ValueOf(v).IsNil() {
				return lua.LNil
			}
			table := state.NewTable()
			//结构体字段
			elem := tp.Elem()
			for i := 0; i < elem.NumField(); i++ {
				field := elem.Field(i)
				if !vp.Elem().IsValid() {
					continue
				}
				value := vp.Elem().Field(i)
				if !field.IsExported() { //非公开的不要
					continue
				}
				key := getFieldKey(field)
				table.RawSetString(key, Marshal(state, value.Interface()))
			}
			if tab, has := RefService(state, t).(*lua.LTable); has {
				tab.ForEach(func(k lua.LValue, v lua.LValue) {
					table.RawSet(k, v)
				})
			}
			return table
		}

		if vp.Elem().Kind() == reflect.Invalid {
			return lua.LNil
		}
		return Marshal(state, vp.Elem().Interface())
	case reflect.Invalid:
		return lua.LNil
	default:
		//fmt.Println("不支持的", vp.Kind())
	}
	return lua.LNil
}

// Unmarshal Lua变量转换为GO变量
func Unmarshal(v any) any {
	if params, has := v.(lua.LValue); has {
		switch params.Type() {
		case lua.LTNil:
			return nil
		case lua.LTBool:
			return params.String()
		case lua.LTNumber:
			return params.String()
		case lua.LTString:
			return params.String()
		case lua.LTTable:
			data := map[string]interface{}{}
			t := params.(*lua.LTable)
			t.ForEach(func(k lua.LValue, v lua.LValue) {
				data[k.String()] = Unmarshal(v)
			})
			return data
		}
		return params.String()
	} else {
		return v
	}
}

// RefService 将IService模块转换成LuaTable方法
func RefService(state *lua.LState, m IService) lua.LValue {
	return refToTable(state, m, func(s string) bool {
		return slices.Contains([]string{"Name"}, s) == false
	})
}

func refToTable(state *lua.LState, v any, filterNameMethods ...func(string) bool) lua.LValue {
	table := state.NewTable()
	rt := reflect.TypeOf(v)
	rv := reflect.ValueOf(v)
	for i := 0; i < rv.NumMethod(); i++ {
		methodType := rt.Method(i)
		methodValue := rv.Method(i)
		mvt := methodValue.Type()
		if !methodValue.IsValid() {
			continue
		}
		if len(filterNameMethods) > 0 {
			if filterNameMethods[0](methodType.Name) == false {
				continue
			}
		}
		callName := lcFirst(methodType.Name)
		table.RawSetString(callName, state.NewFunction(func(state *lua.LState) int {
			defer func() {
				if err := recover(); err != nil {
					log.Println(fmt.Sprintf("调用%s异常: %v", callName, err))
				}
			}()

			inParams := make([]reflect.Value, 0, mvt.NumIn())
			for j := 0; j < mvt.NumIn(); j++ {
				idx := j + 1
				p := mvt.In(j)
				switch p.Kind() {
				case reflect.Bool:
					inParams = append(inParams, toReflectValue(state, p, state.ToBool(idx)))
				case reflect.String,
					reflect.Int,
					reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64,
					reflect.Uint,
					reflect.Uint8,
					reflect.Uint16,
					reflect.Uint32,
					reflect.Uint64,
					reflect.Float32,
					reflect.Float64:
					inParams = append(inParams, toReflectValue(state, p, state.ToString(idx)))
				case reflect.Struct, reflect.Ptr, reflect.Map, reflect.Interface, reflect.Func:
					inParams = append(inParams, toReflectValue(state, p, state.Get(idx)))
				case reflect.Slice:
					//判断当前是否是最后一个入参,如果入参是最后一个,但是lua入参不是最后一个
					if j == mvt.NumIn()-1 {
						////最后一个入参了
						if state.GetTop() == idx { //参数匹配,就一个参数
							//判断是否是table,如果不是table则转换成数组
							if _, h := state.Get(idx).(*lua.LTable); h {
								inParams = append(inParams, toReflectValue(state, p, state.Get(idx)))
							} else {
								inParams = append(inParams, toReflectValue(state, p, []lua.LValue{state.Get(idx)}))
							}
						} else {
							values := make([]lua.LValue, 0, state.GetTop())
							for x := idx; x <= state.GetTop(); x++ {
								values = append(values, state.Get(x))
							}
							inParams = append(inParams, toReflectValue(state, p, values))
						}
					} else {
						inParams = append(inParams, toReflectValue(state, p, state.Get(idx)))
					}
				default:
					inParams = append(inParams, reflect.New(p).Elem()) //创建默认值
				}
			}
			//处理响应结果
			outParams := methodValue.Call(inParams)
			rets := make([]any, 0, len(outParams))
			for _, item := range outParams {
				if item.IsValid() {
					rets = append(rets, item.Interface())
				}
			}
			return ret(state, rets...)
		}))
	}
	return table
}
