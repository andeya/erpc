// Cfgo from the YAML document, bi-directional synchronous multi-module configuration.
//
// The structure of the generated document will reflect the structure of the value itself.
// Maps and pointers (to struct, string, int, etc) are accepted as the in value.
//
// Struct fields are only unmarshalled if they are exported (have an upper case
// first letter), and are unmarshalled using the field name lowercased as the
// default key. Custom keys may be defined via the "yaml" name in the field
// tag: the content preceding the first comma is used as the key, and the
// following comma-separated options are used to tweak the marshalling process.
// Conflicting names result in a runtime error.
//
// The field tag format accepted is:
//
//     `(...) yaml:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The following flags are currently supported:
//
//     omitempty    Only include the field if it's not set to the zero
//                  value for the type or to empty slices or maps.
//                  Does not apply to zero valued structs.
//
//     flow         Marshal using a flow style (useful for structs,
//                  sequences and maps).
//
//     inline       Inline the field, which must be a struct or a map,
//                  causing all of its fields or keys to be processed as if
//                  they were part of the outer struct. For maps, keys must
//                  not conflict with the yaml keys of other struct fields.
//
// In addition, if the key is `-`, the field is ignored.
//
package cfgo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

var (
	defaultCfgo *Cfgo
	defaultInit sync.Once
)

// Default returns default config
func Default() *Cfgo {
	defaultInit.Do(func() {
		defaultCfgo = MustGet("config/config.yaml")
	})
	return defaultCfgo
}

// Filename returns default config file name.
func Filename() string {
	return Default().Filename()
}

// AllowAppsShare allows other applications to share the configuration file.
func AllowAppsShare(allow bool) {
	Default().AllowAppsShare(allow)
}

// MustReg is similar to Reg(), but panic if having error.
func MustReg(section string, structPtr Config) {
	Default().MustReg(section, structPtr)
}

// Reg registers config section to default config file 'config/config.yaml'.
// Automatic callback Reload() to load or reload config.
func Reg(section string, structPtr Config) error {
	return Default().Reg(section, structPtr)
}

// IsReg to determine whether the section is registered.
func IsReg(section string) bool {
	return Default().IsReg(section)
}

// GetSection returns default config section.
func GetSection(section string) (interface{}, bool) {
	return Default().GetSection(section)
}

// BindSection returns default config section copy.
func BindSection(section string, v interface{}) error {
	return Default().BindSection(section, v)
}

// Content returns default yaml config bytes.
func Content() []byte {
	return Default().Content()
}

// ReloadAll reloads all configs.
func ReloadAll() error {
	lock.Lock()
	defer lock.Unlock()
	var errs []string
	for _, c := range cfgos {
		if err := c.Reload(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, string(lineend)))
	}
	return nil
}

// Reload reloads default config.
func Reload() error {
	return Default().Reload()
}

type (
	// Cfgo a whole config
	Cfgo struct {
		filename        string
		originalContent []byte
		content         []byte
		regConfigs      map[string]Config
		extraConfigs    map[string]interface{}
		regSections     Sections
		extraSections   Sections
		allowAppsShare  bool
		lc              sync.RWMutex
	}
	// Config must be struct pointer
	Config interface {
		// load or reload config to app
		Reload(bind BindFunc) error
	}
	// BindFunc bind config to setting
	BindFunc func() error
)

var (
	cfgos   = make(map[string]*Cfgo, 1)
	lock    sync.Mutex
	lineend = func() []byte {
		if runtime.GOOS == "windows" {
			return []byte("\r\n")
		}
		return []byte("\n")
	}()
	indent       = append(lineend, []byte("  ")...)
	dividingLine = append([]byte("# ------------------------- non-automated configuration -------------------------"), lineend...)
)

// MustGet creates a new Cfgo
func MustGet(filename string, allowAppsShare ...bool) *Cfgo {
	c, err := Get(filename, allowAppsShare...)
	if err != nil {
		panic(err)
	}
	return c
}

// Get creates or gets a Cfgo.
func Get(filename string, allowAppsShare ...bool) (*Cfgo, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("[cfgo] %s", err.Error())
	}
	lock.Lock()
	defer lock.Unlock()
	c := cfgos[abs]
	if c != nil {
		return c, nil
	}
	c = &Cfgo{
		filename:        abs,
		originalContent: []byte{},
		content:         []byte{},
		regConfigs:      make(map[string]Config),
		extraConfigs:    make(map[string]interface{}),
		regSections:     make([]*Section, 0, 1),
		extraSections:   make([]*Section, 0),
	}
	if len(allowAppsShare) > 0 && allowAppsShare[0] {
		c.allowAppsShare = true
	}
	err = c.reload()
	if err != nil {
		return nil, fmt.Errorf("[cfgo] %s", err.Error())
	}
	cfgos[abs] = c
	return c, nil
}

// Filename returns the config file name.
func (c *Cfgo) Filename() string {
	return c.filename
}

// AllowAppsShare allows other applications to share the configuration file.
func (c *Cfgo) AllowAppsShare(allow bool) {
	c.allowAppsShare = allow
}

// MustReg is similar to Reg(), but panic if having error.
func (c *Cfgo) MustReg(section string, structPtr Config) {
	err := c.Reg(section, structPtr)
	if err != nil {
		panic(err)
	}
}

// Reg registers config section to config file.
// Automatic callback Reload() to load or reload config.
func (c *Cfgo) Reg(section string, structPtr Config) error {
	c.lc.Lock()
	defer c.lc.Unlock()

	t := reflect.TypeOf(structPtr)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[cfgo] not a struct pointer:\nsection: %s\nstructPtr: %s", section, t.String())
	}
	if s, ok := c.regConfigs[section]; ok {
		return fmt.Errorf("[cfgo] multiple section: %s\nexisted: %s | adding: %s", section, reflect.TypeOf(s).String(), t.String())
	}

	c.regConfigs[section] = structPtr

	// sync config
	var init bool
	var load = func(s string, _ Config, b []byte) error {
		if s == section {
			init = true
			return structPtr.Reload(func() error {
				return yaml.Unmarshal(b, structPtr)
			})
		}
		return nil
	}
	err := c.sync(load)
	if err != nil {
		return err
	}
	if !init {
		err = structPtr.Reload(func() error {
			return nil
		})
	}
	return err
}

// IsReg to determine whether the section is registered.
func (c *Cfgo) IsReg(section string) bool {
	c.lc.RLock()
	defer c.lc.RUnlock()
	_, ok := c.regConfigs[section]
	return ok
}

// GetSection returns yaml config section.
func (c *Cfgo) GetSection(section string) (interface{}, bool) {
	c.lc.RLock()
	defer c.lc.RUnlock()
	if v, ok := c.regConfigs[section]; ok {
		return v, ok
	}
	v, ok := c.extraConfigs[section]
	return v, ok
}

// BindSection returns yaml config section copy.
func (c *Cfgo) BindSection(section string, v interface{}) error {
	c.lc.RLock()
	defer c.lc.RUnlock()
	for _, s := range c.regSections {
		if section == s.title {
			return yaml.Unmarshal(s.single, v)
		}
	}
	for _, s := range c.extraSections {
		if section == s.title {
			return yaml.Unmarshal(s.single, v)
		}
	}
	return fmt.Errorf("not exist config section: %s", section)
}

// Content returns yaml config bytes.
func (c *Cfgo) Content() []byte {
	c.lc.RLock()
	defer c.lc.RUnlock()
	return c.content
}

// Reload reloads config.
func (c *Cfgo) Reload() error {
	c.lc.Lock()
	defer c.lc.Unlock()
	return c.reload()
}

func (c *Cfgo) reload() error {
	return c.sync(func(_ string, setting Config, b []byte) error {
		return setting.Reload(func() error {
			return yaml.Unmarshal(b, setting)
		})
	})
}

func (c *Cfgo) clean() {
	c.originalContent = c.originalContent[:0]
	c.content = c.content[:0]
	c.extraConfigs = make(map[string]interface{})
	c.regSections = c.regSections[:0]
	c.extraSections = c.extraSections[:0]
}

func (c *Cfgo) sync(load func(section string, setting Config, b []byte) error) (err error) {
	c.clean()
	defer func() {
		if err != nil {
			err = fmt.Errorf("[cfgo] %s", err.Error())
		}
	}()
	d, _ := filepath.Split(c.filename)
	err = os.MkdirAll(d, 0777)
	if err != nil {
		return
	}

	// unmarshal
	err = c.read(load)
	if err != nil {
		return
	}

	// Restore the original configuration
	defer func() {
		if err != nil {
			file, err := os.OpenFile(c.filename, os.O_WRONLY|os.O_SYNC|os.O_TRUNC|os.O_CREATE, 0666)
			if err != nil {
				return
			}
			defer file.Close()
			file.Write(c.originalContent)
		}
	}()

	// marshal
	err = c.write()
	if err != nil {
		return
	}

	return nil
}

func (c *Cfgo) read(load func(section string, setting Config, b []byte) error) (err error) {
	file, err := os.OpenFile(c.filename, os.O_RDONLY|os.O_SYNC|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	c.originalContent, err = ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		return
	}

	err = yaml.Unmarshal(c.originalContent, &c.extraConfigs)
	if err != nil {
		return
	}

	// load config
	var errs []string
	var single []byte
	for k, v := range c.extraConfigs {
		for kk, vv := range c.regConfigs {
			if k == kk {
				delete(c.extraConfigs, k)
				if single, err = yaml.Marshal(v); err != nil {
					return
				}
				// load
				if err = load(kk, vv, single); err != nil {
					errs = append(errs, err.Error())
				}
				break
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, string(lineend)))
	}

	var section *Section
	c.regSections = make([]*Section, 0, len(c.regConfigs))
	for k, v := range c.regConfigs {
		if section, err = createSection(k, v); err != nil {
			return
		}
		c.regSections = append(c.regSections, section)
	}
	sort.Sort(c.regSections)

	c.extraSections = make([]*Section, 0, len(c.extraConfigs))
	for k, v := range c.extraConfigs {
		if section, err = createSection(k, v); err != nil {
			return
		}
		c.extraSections = append(c.extraSections, section)
	}
	sort.Sort(c.extraSections)
	return nil
}

func createSection(k string, v interface{}) (section *Section, err error) {
	section = &Section{
		title: k,
	}
	var single []byte
	if single, err = yaml.Marshal(v); err != nil {
		return
	}
	section.single = single
	var united = bytes.Replace(single, []byte("\r\n"), []byte("\n"), -1)
	united = bytes.Replace(united, []byte("\n"), indent, -1)
	united = append([]byte(k+":"+string(lineend)+"  "), united[:len(united)-2]...)
	section.united = united
	return
}

func (c *Cfgo) write() error {
	file, err := os.OpenFile(c.filename, os.O_WRONLY|os.O_SYNC|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	content := bytes.NewBuffer(c.content)
	w := io.MultiWriter(file, content)

	if c.allowAppsShare {
		// Allow multiple processes share

		allSections := append(c.regSections, c.extraSections...)
		sort.Sort(allSections)
		for i, section := range allSections {
			if i != 0 {
				_, err = w.Write(lineend)
				if err != nil {
					return err
				}
			}
			_, err = w.Write(section.united)
			if err != nil {
				return err
			}
		}
	} else {
		// Only single process

		for i, section := range c.regSections {
			if i != 0 {
				_, err = w.Write(lineend)
				if err != nil {
					return err
				}
			}
			_, err = w.Write(section.united)
			if err != nil {
				return err
			}
		}
		for i, section := range c.extraSections {
			if i == 0 {
				_, err = w.Write(lineend)
				if err != nil {
					return err
				}
				_, err = w.Write(dividingLine)
				if err != nil {
					return err
				}
			}
			_, err = w.Write(lineend)
			if err != nil {
				return err
			}
			_, err = w.Write(section.united)
			if err != nil {
				return err
			}
		}
	}

	c.content = content.Bytes()
	return nil
}

type (
	Sections []*Section
	Section  struct {
		title  string
		single []byte
		united []byte
	}
)

// Len is the number of elements in the collection.
func (s Sections) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s Sections) Less(i, j int) bool {
	return s[i].title < s[j].title
}

// Swap swaps the elements with indexes i and j.
func (s Sections) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
