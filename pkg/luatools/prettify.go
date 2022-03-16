package luatools

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

const eol = "\r\n"

type DataPrettify struct {
	vm   *lua.LState
	data interface{}
	name string
}

func NewDataPrettify() *DataPrettify {
	return &DataPrettify{
		vm: lua.NewState(lua.Options{
			MinimizeStackMemory: true,
		}),
	}
}

func (d *DataPrettify) Load(filename string) error {
	abspath, err := filepath.Abs(filename)
	if err != nil {
		abspath = filename
	}
	b, err := os.ReadFile(abspath)
	if err != nil {
		return err
	}
	if err := d.vm.DoString(string(b)); err != nil {
		return err
	}
	name := filepath.Base(filename)
	name = strings.ReplaceAll(name, filepath.Ext(name), "")
	name = regexp.MustCompile("[^A-Z|a-z]").ReplaceAllString(name, "")
	data := d.vm.GetGlobal("Data").(*lua.LTable).RawGet(lua.LString(name))
	if data == lua.LNil {
		return fmt.Errorf("key %v is NilType", name)
	} else if _, ok := data.(*lua.LTable); !ok {
		return fmt.Errorf("value type mismatched")
	}
	d.data = d.decode(data)
	d.name = name
	return nil
}

func (d *DataPrettify) WriteToFile(filename string) error {
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	dict := d.data.(map[interface{}]interface{})
	keys := make([]interface{}, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		switch keys[i].(type) {
		case int:
			return keys[i].(int) < keys[j].(int)
		case string:
			return keys[i].(string) < keys[j].(string)
		}
		return false
	})
	b := bytes.Buffer{}
	for _, k := range keys {
		v := dict[k]
		ctx := d.stringify(v)
		var value string
		switch k.(type) {
		case int:
			value = fmt.Sprintf("[%v]=%v,"+eol, k, ctx)
		case string:
			value = fmt.Sprintf("%v=%v,"+eol, k, ctx)
		}
		b.WriteString(value)
	}
	f.WriteString("-- $Id$" + eol)
	f.WriteString(eol)
	f.WriteString("module(\"Data\")" + eol)
	f.WriteString(eol)
	f.WriteString(fmt.Sprintf("%v="+eol, d.name))
	f.WriteString("{")
	f.WriteString(eol)
	f.Write(d.pretty(b.Bytes()))
	f.WriteString("}")
	return nil
}

func (d *DataPrettify) isInteger(val float64) bool {
	return val == float64(int(val))
}

func (d *DataPrettify) stringify(value interface{}) string {
	var ctx string
	switch value.(type) {
	case float64:
		ctx += fmt.Sprintf("%f", value.(float64))
	case int64:
		ctx += fmt.Sprintf("%v", value.(int64))
	case bool:
		if value.(bool) {
			ctx += "true"
		} else {
			ctx += "false"
		}
	case string:
		ctx += fmt.Sprintf("\"%v\"", value.(string))
	case map[interface{}]interface{}:
		tbl := value.(map[interface{}]interface{})
		ctx += "{"
		keys := make([]interface{}, 0, len(tbl))
		for k := range tbl {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			switch keys[i].(type) {
			case int:
				return keys[i].(int) < keys[j].(int)
			case string:
				return keys[i].(string) < keys[j].(string)
			}
			return false
		})
		if len(keys) > 0 {
			if len(keys) == keys[len(keys)-1] {
				for _, k := range keys {
					ctx += d.stringify(tbl[k]) + ","
				}
			} else {
				for _, k := range keys {
					v := tbl[k]
					switch k.(type) {
					case int:
						ctx += fmt.Sprintf("[%v] = ", k)
					case bool:
						ctx += fmt.Sprintf("%v = ", k)
					case string:
						ctx += fmt.Sprintf("%v = ", k)
					}
					ctx += d.stringify(v) + ","
				}
			}
		}
		ctx += "}"
	}
	return ctx
}

func (d *DataPrettify) padding(n int) string {
	var paddings string
	for i := 0; i < n; i++ {
		paddings += "\t"
	}
	return paddings
}

func (d *DataPrettify) pretty(b []byte) []byte {
	depth := 1
	swap := bytes.Buffer{}
	r := bufio.NewReader(bytes.NewReader(b))
	var err error
	for err != io.EOF {
		var line string
		line, err = r.ReadString('\n')
		line = strings.Trim(line, "\n")
		line = strings.Trim(line, "\r")
		line = strings.Trim(line, "\t")
		line = strings.TrimSpace(line)
		var o string
		o += d.padding(depth)
		for _, s := range line {
			switch s {
			case '{':
				o += eol
				o += d.padding(depth)
				depth += 1
			case '}':
				o += eol
				depth -= 1
				o += d.padding(depth)
			}
			o += string(s)
			switch s {
			case ',':
				o += eol
				o += d.padding(depth)
			case '{':
				o += eol
				o += d.padding(depth)
			}
		}
		if len(o) > 0 {
			swap.WriteString(o)
		}
	}
	m := regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)
	return m.ReplaceAll(swap.Bytes(), []byte{})
}

func (d *DataPrettify) decode(value lua.LValue) interface{} {
	var ret interface{}
	switch value.Type() {
	case lua.LTString:
		ret = value.String()
	case lua.LTBool:
		ret = bool(value.(lua.LBool))
	case lua.LTNumber:
		f := float64(value.(lua.LNumber))
		if d.isInteger(f) {
			ret = int64(value.(lua.LNumber))
		} else {
			ret = float64(value.(lua.LNumber))
		}
	case lua.LTTable:
		if tbl, ok := value.(*lua.LTable); ok {
			dict := make(map[interface{}]interface{})
			tbl.ForEach(func(k lua.LValue, v lua.LValue) {
				var key interface{}
				switch k.Type() {
				case lua.LTNumber:
					key = int(k.(lua.LNumber))
				case lua.LTBool:
					key = bool(k.(lua.LBool))
				case lua.LTString:
					key = k.String()
				}
				dict[key] = d.decode(v)
			})
			ret = dict
		}
	}
	return ret
}
