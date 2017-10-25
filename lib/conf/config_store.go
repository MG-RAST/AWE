package conf

import (
	"flag"
	"fmt"
	"github.com/MG-RAST/golib/goconfig/config"
	"os"
)

type Config_store struct {
	Store []*Config_value
	Fs    *flag.FlagSet
	Con   *config.Config
}

type Config_value struct {
	Conf_type string
	Conf_str  *Config_value_string
	Conf_int  *Config_value_int
	Conf_bool *Config_value_bool
}

type Config_value_string struct {
	Target        *string
	Default_value string
	Section       string
	Key           string
	Descr_short   string
	Descr_long    string
}

type Config_value_int struct {
	Target        *int
	Default_value int
	Section       string
	Key           string
	Descr_short   string
	Descr_long    string
}

type Config_value_bool struct {
	Target        *bool
	Default_value bool
	Section       string
	Key           string
	Descr_short   string
	Descr_long    string
}

func NewCS(c *config.Config) *Config_store {
	cs := &Config_store{Store: make([]*Config_value, 0, 100), Con: c} // length 0, capacity 100
	cs.Fs = flag.NewFlagSet("name", flag.ContinueOnError)
	cs.Fs.BoolVar(&FAKE_VAR, "fake_var", true, "ignore this")
	cs.Fs.BoolVar(&SHOW_HELP, "h", false, "ignore this") // for help: -h
	return cs
}

func (this *Config_store) AddString(target *string,
	default_value string,
	section string,
	key string,
	descr_short string,
	descr_long string) {

	*target = default_value
	new_val := &Config_value{Conf_type: "string"}
	new_val.Conf_str = &Config_value_string{target, default_value, section, key, descr_short, descr_long}

	this.Store = append(this.Store, new_val)
}

func (this *Config_store) AddInt(target *int,
	default_value int,
	section string,
	key string,
	descr_short string,
	descr_long string) {

	*target = default_value
	new_val := &Config_value{Conf_type: "int"}
	new_val.Conf_int = &Config_value_int{target, default_value, section, key, descr_short, descr_long}

	this.Store = append(this.Store, new_val)
}

func (this *Config_store) AddBool(target *bool,
	default_value bool,
	section string,
	key string,
	descr_short string,
	descr_long string) {

	*target = default_value
	new_val := &Config_value{Conf_type: "bool"}
	new_val.Conf_bool = &Config_value_bool{target, default_value, section, key, descr_short, descr_long}

	this.Store = append(this.Store, new_val)
}

func (this Config_store) Parse() {
	c := this.Con
	f := this.Fs
	for _, val := range this.Store {
		if val.Conf_type == "string" {
			get_my_config_string(c, f, val.Conf_str)
		} else if val.Conf_type == "int" {
			get_my_config_int(c, f, val.Conf_int)
		} else if val.Conf_type == "bool" {
			get_my_config_bool(c, f, val.Conf_bool)
		}
	}
	err := this.Fs.Parse(os.Args[1:])
	if err != nil {
		this.PrintHelp()
		fmt.Fprintf(os.Stderr, "error parsing command line args: "+err.Error()+"\n")
		os.Exit(1)
	}
}

func (this Config_store) PrintHelp() {
	current_section := ""
	prefix := "--"
	if PRINT_HELP {
		prefix = ""
	}
	for _, val := range this.Store {
		if val.Conf_type == "string" {
			d := val.Conf_str
			if current_section != d.Section {
				current_section = d.Section
				fmt.Printf("\n[%s]\n", current_section)
			}
			fmt.Printf("%s%-27s %s (default: \"%s\")\n", prefix, d.Key+"=<string>", d.Descr_short, d.Default_value)

			if PRINT_HELP && d.Descr_long != "" {
				fmt.Printf("     %s\n", d.Descr_long)
			}
		} else if val.Conf_type == "int" {
			d := val.Conf_int
			if current_section != d.Section {
				current_section = d.Section
				fmt.Printf("\n[%s]\n", current_section)
			}
			fmt.Printf("%s%-27s %s (default: %d)\n", prefix, d.Key+"=<int>", d.Descr_short, d.Default_value)

			if PRINT_HELP && d.Descr_long != "" {
				fmt.Printf("     %s\n", d.Descr_long)
			}
		} else if val.Conf_type == "bool" {
			d := val.Conf_bool
			if current_section != d.Section {
				current_section = d.Section
				fmt.Printf("\n[%s]\n", current_section)
			}
			fmt.Printf("%s%-27s %s (default: %t)\n", prefix, d.Key+"=<bool>", d.Descr_short, d.Default_value)

			if PRINT_HELP && d.Descr_long != "" {
				fmt.Printf("     %s\n", d.Descr_long)
			}
		}
	}

}

// writes to target only if has been defined in config
// avoids overwriting of default values if config is not defined
func getDefinedValueInt(c *config.Config, section string, key string, target *int) {
	if c.HasOption(section, key) {
		if int_value, err := c.Int(section, key); err == nil {
			*target = int_value
		}
	}
}

func getDefinedValueBool(c *config.Config, section string, key string, target *bool) {
	if c.HasOption(section, key) {
		if bool_value, err := c.Bool(section, key); err == nil {
			*target = bool_value
		}
	}
}

func getDefinedValueString(c *config.Config, section string, key string, target *string) {
	if string_value, err := c.String(section, key); err == nil {
		string_value = os.ExpandEnv(string_value)
		*target = string_value
	}

}
