package luatools

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type DataComparism struct {
	vm        *lua.LState
	values    []lua.LValue
	Entry     string
	Checksums []string
}

func NewDataComparism() *DataComparism {
	return &DataComparism{
		Checksums: make([]string, 2),
		values:    make([]lua.LValue, 2),
		vm: lua.NewState(lua.Options{
			MinimizeStackMemory: true,
		}),
	}
}

func (c *DataComparism) Load(filenames []string) error {
	entries := make([]string, 0)
	const module = `module("Data")`
	for i, filename := range filenames {
		abspath, _ := filepath.Abs(filename)
		b, err := os.ReadFile(abspath)
		if err != nil {
			return err
		}
		c.Checksums[i] = fmt.Sprintf("%x", md5.Sum(b))
		f := bytes.Buffer{}
		r := bufio.NewReader(bytes.NewBuffer(b))
		for err != io.EOF {
			var line string
			line, err = r.ReadString('\n')
			if strings.Contains(line, "md5sum") {
				continue
			} else if strings.Contains(line, module) {
				newmodule := fmt.Sprintf("module(\"Data%v\")", i)
				line = strings.ReplaceAll(line, module, newmodule)
			}
			f.WriteString(line)
		}
		if err := c.vm.DoString(f.String()); err != nil {
			return err
		}
		data, ok := c.vm.GetGlobal(fmt.Sprintf("Data%v", i)).(*lua.LTable)
		if !ok {
			return fmt.Errorf("global variable Data%v is not a table", i)
		}
		var entry string
		data.ForEach(func(l1, l2 lua.LValue) {
			if strings.HasPrefix(l1.String(), "_") {
				return
			} else if len(entry) > 0 {
				return
			}
			entry = l1.String()
		})
		if len(entry) == 0 {
			continue
		}
		entries = append(entries, entry)
		c.values[i] = data.RawGet(lua.LString(entry))
	}
	if len(entries) < 2 {
		return fmt.Errorf("entries is less than two %v", entries)
	} else if entries[0] != entries[len(entries)-1] {
		return fmt.Errorf("entries mismatched %v", entries)
	}
	c.Entry = entries[len(entries)-1]
	return nil
}

func (c *DataComparism) Equal() (bool, error) {
	script := `
function deepcompare(table1, table2)
	local avoid_loops = {}
	local function recurse(t1, t2)
	   -- compare value types
	   if type(t1) ~= type(t2) then return false end
	   -- Base case: compare simple values
	   if type(t1) ~= "table" then return t1 == t2 end
	   -- Now, on to tables.
	   -- First, let's avoid looping forever.
	   if avoid_loops[t1] then return avoid_loops[t1] == t2 end
	   avoid_loops[t1] = t2
	   -- Copy keys from t2
	   local t2keys = {}
	   local t2tablekeys = {}
	   for k, _ in pairs(t2) do
		  if type(k) == "table" then table.insert(t2tablekeys, k) end
		  t2keys[k] = true
	   end
	   -- Let's iterate keys from t1
	   for k1, v1 in pairs(t1) do
		  local v2 = t2[k1]
		  if type(k1) == "table" then
			 -- if key is a table, we need to find an equivalent one.
			 local ok = false
			 for i, tk in ipairs(t2tablekeys) do
				if deepcompare(k1, tk) and recurse(v1, t2[tk]) then
				   table.remove(t2tablekeys, i)
				   t2keys[tk] = nil
				   ok = true
				   break
				end
			 end
			 if not ok then return false end
		  else
			 -- t1 has a key which t2 doesn't have, fail.
			 if v2 == nil then return false end
			 t2keys[k1] = nil
			 if not recurse(v1, v2) then return false end
		  end
	   end
	   -- if t2 has a key which t1 doesn't have, fail.
	   if next(t2keys) then return false end
	   return true
	end
	return recurse(table1, table2)
end
	`
	if err := c.vm.DoString(script); err != nil {
		return false, err
	}
	if err := c.vm.CallByParam(lua.P{
		Fn:      c.vm.GetGlobal("deepcompare"),
		NRet:    1,
		Protect: true,
	}, c.values...); err != nil {
		panic(err)
	}
	ret := c.vm.Get(-1)
	defer c.vm.Pop(1)
	if _, ok := ret.(lua.LBool); !ok {
		return false, errors.New("logic error")
	}
	return ret == lua.LTrue, nil
}

func (c *DataComparism) Close() {
	c.vm.Close()
}
