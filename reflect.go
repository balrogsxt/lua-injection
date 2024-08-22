package lua_env

import (
	lua "github.com/yuin/gopher-lua"
	"log"
	"reflect"
)

func toReflectValue(state *lua.LState, t reflect.Type, v any) reflect.Value {
	switch t.Kind() {
	case reflect.Interface:
		return reflect.ValueOf(Unmarshal(v))
	case reflect.String:
		return reflect.ValueOf(NewVar(v).String())
	case reflect.Int:
		return reflect.ValueOf(NewVar(v).Int())
	case reflect.Int8:
		return reflect.ValueOf(NewVar(v).Int8())
	case reflect.Int16:
		return reflect.ValueOf(NewVar(v).Int16())
	case reflect.Int32:
		return reflect.ValueOf(NewVar(v).Int32())
	case reflect.Int64:
		return reflect.ValueOf(NewVar(v).Int64())
	case reflect.Uint:
		return reflect.ValueOf(NewVar(v).Uint())
	case reflect.Uint8:
		return reflect.ValueOf(NewVar(v).Uint8())
	case reflect.Uint16:
		return reflect.ValueOf(NewVar(v).Uint16())
	case reflect.Uint32:
		return reflect.ValueOf(NewVar(v).Uint32())
	case reflect.Uint64:
		return reflect.ValueOf(NewVar(v).Uint64())
	case reflect.Float32:
		return reflect.ValueOf(NewVar(v).Float32())
	case reflect.Float64:
		return reflect.ValueOf(NewVar(v).Float64())
	case reflect.Bool:
		return reflect.ValueOf(NewVar(v).Bool())
	case reflect.Ptr:
		ov := toReflectValue(state, t.Elem(), v)
		nv := reflect.New(ov.Type())
		if v == nil || reflect.TypeOf(&lua.LNilType{}) == reflect.TypeOf(v) {
			nv.Elem().SetZero()
		} else {
			nv.Elem().Set(ov)
		}
		return nv
	case reflect.Struct:
		r := reflect.New(t).Elem()
		tab, has := v.(*lua.LTable)
		if !has {
			return r
		}
		for i := 0; i < r.NumField(); i++ {
			ft := r.Type().Field(i)
			fv := r.Field(i)
			if !ft.IsExported() { //非公开的不要
				continue
			}
			key := getFieldKey(ft)
			tv := tab.RawGetString(key)
			if tv != lua.LNil {
				switch fv.Kind() {
				case reflect.String:
					fv.SetString(tv.String())
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					fv.SetInt(NewVar(tv.String()).Int64())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					fv.SetUint(NewVar(tv.String()).Uint64())
				case reflect.Float32, reflect.Float64:
					fv.SetFloat(NewVar(tv.String()).Float64())
				case reflect.Struct, reflect.Slice, reflect.Map, reflect.Ptr:
					fv.Set(toReflectValue(state, ft.Type, tv))
				case reflect.Invalid:
					fv.SetZero()
				default:
				}
			} else {
				fv.SetZero()
			}

		}
		return r
	case reflect.Func: //回调函数类型
		fn, has := v.(*lua.LFunction)
		if !has {
			return reflect.ValueOf(nil)
		}
		return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
			//获取实际go定义的入参类型和返参类型
			inParams := make([]lua.LValue, 0)
			for j := 0; j < t.NumIn(); j++ {
				item := args[j]
				inParams = append(inParams, Marshal(state, item.Interface()))
			}
			co, _ := state.NewThread()
			_, err, rt := state.Resume(co, fn, inParams...)
			//将lua变量转换成go变量
			outParams := make([]reflect.Value, 0)
			for j := 0; j < t.NumOut(); j++ {
				p := t.Out(j)
				if len(rt) > j {
					item := rt[j]
					switch p.Kind() {
					case reflect.String, reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Bool:
						outParams = append(outParams, toReflectValue(state, p, item.String()))
					case reflect.Struct, reflect.Ptr, reflect.Map, reflect.Interface, reflect.Slice, reflect.Func:
						outParams = append(outParams, toReflectValue(state, p, item))
					default:
						outParams = append(outParams, reflect.New(p).Elem())
					}
				} else {
					outParams = append(outParams, reflect.New(p).Elem())
				}
			}
			if err != nil {
				return outParams
			}
			return outParams
		})

	case reflect.Map:
		mapVal := reflect.MakeMap(t)
		tab, has := v.(*lua.LTable)
		if !has {
			return mapVal
		}
		tab.ForEach(func(key lua.LValue, val lua.LValue) {
			mapVal.SetMapIndex(toReflectValue(state, t.Key(), key), toReflectValue(state, t.Elem(), val))
		})
		return mapVal
	case reflect.Slice:
		tab, has := v.(*lua.LTable)
		if !has {
			lvs, lvh := v.([]lua.LValue)
			if lvh {
				slice := reflect.MakeSlice(t, 0, 0)
				for _, item := range lvs {
					switch t.Elem().Kind() {
					case reflect.String, reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Bool:
						slice = reflect.Append(slice, toReflectValue(state, t.Elem(), item.String()))
					default:
						slice = reflect.Append(slice, toReflectValue(state, t.Elem(), item))
					}
				}
				return slice
			}
			return reflect.ValueOf(v)
		}
		sliceVal := reflect.MakeSlice(t, 0, 0)
		tab.ForEach(func(_ lua.LValue, val lua.LValue) {
			tv, vh := val.(*lua.LTable)
			var vv any = val.String()
			if vh {
				vv = tv
			}
			sliceVal = reflect.Append(sliceVal, toReflectValue(state, t.Elem(), vv))
		})
		return sliceVal
	default:
		log.Println("暂未支持的转换类型: ", t.String())
		return reflect.New(t).Elem()
	}
}
