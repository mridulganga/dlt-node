package plugin

import (
	errorsPkg "errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"sort"

	"github.com/pkg/errors"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// L is the lua state
// This is the VM that runs the plugins

type PluginManager struct {
	L       *lua.LState
	plugins map[string]Plugin
}

// Plugin ...
type Plugin struct {
	Path string
	Name string
}

func NewPluginManager() PluginManager {
	return PluginManager{
		L:       lua.NewState(),
		plugins: map[string]Plugin{},
	}
}

// Each Call clb with each plugins sorted by name
func (pm *PluginManager) Each(clb func(p Plugin)) {
	var loadedPlugins []Plugin
	for _, v := range pm.plugins {
		loadedPlugins = append(loadedPlugins, v)
	}
	sort.Slice(loadedPlugins, func(i, j int) bool { return loadedPlugins[i].Name < loadedPlugins[j].Name })
	for _, v := range loadedPlugins {
		clb(v)
	}
}

// Set a global variable in lua VM
func (pm *PluginManager) Set(name string, val interface{}) {
	pm.L.SetGlobal(name, luar.New(pm.L, val))
}

func (pm *PluginManager) SetBulk(funcs map[string]any) {
	for k, v := range funcs {
		pm.Set(k, v)
	}
}

// IsLoaded returns whether the file is loaded or not in the lua VM
func (pm *PluginManager) IsLoaded(path string) bool {
	filePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	_, ok := pm.plugins[filePath]
	return ok
}

// Unload remove plugin functions from lua VM
func (pm *PluginManager) Unload(path string) error {
	filePath, _ := filepath.Abs(path)
	_, fileName := filepath.Split(filePath)
	fileExt := filepath.Ext(fileName)
	pluginName := strings.TrimSuffix(fileName, fileExt)
	str := "\nlocal P = {}\n" + pluginName + " = P\nsetmetatable(" + pluginName + ", {__index = _G})\nsetfenv(1, P)\n"
	if err := pm.L.DoString(str); err != nil {
		return errors.Wrap(err, "Unload: Failed to unload plugin")
	}
	delete(pm.plugins, filePath)
	return nil
}

// Load plugin functions in lua VM using plugin contents
func (pm *PluginManager) LoadFromString(pluginName, pluginData string) (Plugin, error) {
	var newPlugin Plugin
	pluginDef := "\nlocal P = {}\n" + pluginName + " = P\nsetmetatable(" + pluginName + ", {__index = _G})\nsetfenv(1, P)\n"
	if err := pm.L.DoString(pluginDef + string(pluginData)); err != nil {
		return newPlugin, errors.Wrap(err, "Load: Unable to execute lua string")
	}
	newPlugin = Plugin{pluginName, pluginName}
	pm.plugins[pluginName] = newPlugin
	return newPlugin, nil
}

// Load plugin functions in lua VM using plugin file
func (pm *PluginManager) LoadFile(path string) (Plugin, error) {
	var newPlugin Plugin
	filePath, _ := filepath.Abs(path)
	_, fileName := filepath.Split(filePath)
	fileExt := filepath.Ext(fileName)
	pluginName := strings.TrimSuffix(fileName, fileExt)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return newPlugin, errors.Wrap(err, fmt.Sprintf("Load: Unable to read file %s", filePath))
	}
	pluginDef := "\nlocal P = {}\n" + pluginName + " = P\nsetmetatable(" + pluginName + ", {__index = _G})\nsetfenv(1, P)\n"
	if err := pm.L.DoString(pluginDef + string(data)); err != nil {
		return newPlugin, errors.Wrap(err, "Load: Unable to execute lua string")
	}
	newPlugin = Plugin{filePath, pluginName}
	pm.plugins[filePath] = newPlugin
	return newPlugin, nil
}

// Call a function in the lua VM
func (pm *PluginManager) Call(fn string, args ...interface{}) (lua.LValue, error) {
	var luaFunc lua.LValue
	if strings.Contains(fn, ".") {
		plugin := pm.L.GetGlobal(strings.Split(fn, ".")[0])
		if plugin.String() == "nil" {
			return nil, errors.New("function does not exist: " + fn)
		}
		luaFunc = pm.L.GetField(plugin, strings.Split(fn, ".")[1])
	} else {
		luaFunc = pm.L.GetGlobal(fn)
	}
	if luaFunc.String() == "nil" {
		return nil, errors.New("function does not exist: " + fn)
	}
	var luaArgs []lua.LValue
	for _, v := range args {
		luaArgs = append(luaArgs, luar.New(pm.L, v))
	}
	err := pm.L.CallByParam(lua.P{
		Fn:      luaFunc,
		NRet:    1,
		Protect: true,
	}, luaArgs...)
	if err != nil {
		return nil, err
	}
	ret := pm.L.Get(-1) // returned value
	pm.L.Pop(1)         // remove received value
	return ret, nil
}

func (pm *PluginManager) CallUnmarshal(v interface{}, fn string, args ...interface{}) error {
	rv := reflect.ValueOf(v)
	nv := reflect.Indirect(rv)
	lv, err := pm.Call(fn, args...)
	switch v.(type) {
	case *string:
		if str, ok := lv.(lua.LString); ok {
			nv.SetString(string(str))
		}
	case *int:
		if nb, ok := lv.(lua.LNumber); ok {
			nv.SetInt(int64(nb))
		}
	case *bool:
		if b, ok := lv.(lua.LBool); ok {
			nv.SetBool(bool(b))
		}
	case *map[string]string:
		if b, ok := lv.(*lua.LTable); ok {
			b.ForEach(func(x lua.LValue, y lua.LValue) {
				xValue := reflect.ValueOf(x)
				yValue := reflect.ValueOf(y)
				nv.SetMapIndex(xValue, yValue)
			})
		}
	case *map[string]any:
		if b, ok := lv.(*lua.LTable); ok {
			b.ForEach(func(x lua.LValue, y lua.LValue) {
				xValue := reflect.ValueOf(x)
				yValue := reflect.ValueOf(y)
				nv.SetMapIndex(xValue, yValue)
			})
		}
	case *map[any]any:
		if b, ok := lv.(*lua.LTable); ok {
			b.ForEach(func(x lua.LValue, y lua.LValue) {
				xValue := reflect.ValueOf(x)
				yValue := reflect.ValueOf(y)
				nv.SetMapIndex(xValue, yValue)
			})
		}
	default:
		err = errorsPkg.New("invalid type")
	}

	return err
}

// Close lua VM
func (pm *PluginManager) Close() {
	pm.L.Close()
}
