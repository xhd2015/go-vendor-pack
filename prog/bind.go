package prog

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Bind(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		panic(fmt.Errorf("requires ptr, actual: %T", v))
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		panic(fmt.Errorf("requires struct, actual: %T", v))
	}

	t := rv.Type()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		fieldValue := rv.Field(i)
		field := t.Field(i)

		if field.Anonymous {
			// recursively parse
			Bind(fieldValue.Addr().Interface())
			continue
		}

		s := field.Tag.Get("prog")
		if s == "" {
			continue
		}

		list := strings.SplitN(s, " ", 3)
		if len(list) != 3 {
			panic(fmt.Errorf("invalid option: %v, require 3 parts,actual: %d", field.Name, len(list)))
		}
		flagName := list[0]
		defaulVal := list[1]
		help := list[2]

		var bad bool
		switch fieldValue.Kind() {
		case reflect.String:
			if defaulVal == "''" {
				defaulVal = ""
			}
			flag.StringVar(fieldValue.Addr().Interface().(*string), flagName, defaulVal, help)
		case reflect.Bool:
			v, err := strconv.ParseBool(defaulVal)
			if err != nil {
				panic(fmt.Errorf("parsing %s as bool: invalid default value %s", field.Name, defaulVal))
			}
			flag.BoolVar(fieldValue.Addr().Interface().(*bool), flagName, v, help)
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.String {
				// []string
				plist := fieldValue.Addr().Interface().(*[]string)
				flag.Var(strSlice{plist}, flagName, help)
			} else {
				bad = true
			}
		default:
			bad = true
		}
		if bad {
			panic(fmt.Errorf("unsupported type: %s %s", field.Name, fieldValue.Type()))
		}
	}
}

type strSlice struct {
	ptr *[]string
}

func (c strSlice) String() string {
	return strings.Join(*c.ptr, ",")
}

func (c strSlice) Set(value string) error {
	*c.ptr = append(*c.ptr, value)
	return nil
}
