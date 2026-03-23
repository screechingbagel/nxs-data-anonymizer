package ctx

import (
	"fmt"

	"github.com/nixys/nxs-data-anonymizer/misc"
	conf "github.com/nixys/nxs-go-conf"
)

type ConfOpts struct {
	LogFile  string `conf:"logfile" conf_extraopts:"default=stderr"`
	LogLevel string `conf:"loglevel" conf_extraopts:"default=info"`

	Progress  ProgressConf                  `conf:"progress"`
	Filters   map[string]FilterConf         `conf:"filters"`
	Link      []LinkConf                    `conf:"link"`
	Security  SecurityConf                  `conf:"security"`
	Variables map[string]VariableFilterConf `conf:"variables"`

	MySQL *MySQLConf `conf:"mysql"`
}

type ProgressConf struct {
	Rhythm   string `conf:"rhythm" conf_extraopts:"default=0s"`
	Humanize bool   `conf:"humanize"`
}

type FilterConf struct {
	Columns map[string]ColumnFilterConf `conf:"columns"`
}

type ColumnFilterConf struct {
	Type   string `conf:"type" conf_extraopts:"default=template"`
	Value  string `conf:"value" conf_extraopts:"required"`
	Unique bool   `conf:"unique"`
}

type LinkConf struct {
	Rule ColumnFilterConf    `conf:"rule"`
	With map[string][]string `conf:"with" conf_extraopts:"required"`
}

type VariableFilterConf struct {
	Type  string `conf:"type" conf_extraopts:"default=template"`
	Value string `conf:"value" conf_extraopts:"required"`
}

type SecurityConf struct {
	Policy     SecurityPolicyConf     `conf:"policy"`
	Exceptions SecurityExceptionsConf `conf:"exceptions"`
	Defaults   SecurityDefaultsConf   `conf:"defaults"`
}

type SecurityPolicyConf struct {
	Tables  string `conf:"tables" conf_extraopts:"default=pass"`
	Columns string `conf:"columns" conf_extraopts:"default=pass"`
}

type SecurityExceptionsConf struct {
	Tables  []string `conf:"tables"`
	Columns []string `conf:"columns"`
}

type SecurityDefaultsConf struct {
	Columns map[string]ColumnFilterConf `conf:"columns"`
	Types   []SecurityDefaultsTypeConf  `conf:"types"`
}

type SecurityDefaultsTypeConf struct {
	Regex string           `conf:"regex" conf_extraopts:"required"`
	Rule  ColumnFilterConf `conf:"rule" conf_extraopts:"required"`
}

type MySQLConf struct {
	Host     string `conf:"host" conf_extraopts:"required"`
	Port     int    `conf:"port" conf_extraopts:"required"`
	DB       string `conf:"db" conf_extraopts:"required"`
	User     string `conf:"user" conf_extraopts:"required"`
	Password string `conf:"password" conf_extraopts:"required"`
}

func ConfRead(confPath string) (ConfOpts, error) {

	var c ConfOpts

	err := conf.Load(&c, conf.Settings{
		ConfPath:    confPath,
		ConfType:    conf.ConfigTypeYAML,
		UnknownDeny: true,
	})
	if err != nil {
		return c, err
	}

	for _, f := range c.Filters {
		for _, cf := range f.Columns {
			if misc.ValueTypeFromString(cf.Type) == misc.ValueTypeUnknown {
				return c, fmt.Errorf("conf read: unknown column filter type")
			}
		}
	}

	for _, f := range c.Variables {
		if misc.ValueTypeFromString(f.Type) == misc.ValueTypeUnknown {
			return c, fmt.Errorf("conf read: unknown variable filter type")
		}
	}

	if misc.SecurityPolicyTablesTypeFromString(c.Security.Policy.Tables) == misc.SecurityPolicyTablesUnknown {
		return c, fmt.Errorf("conf read: unknown security policy tables type")
	}

	if misc.SecurityPolicyColumnsTypeFromString(c.Security.Policy.Columns) == misc.SecurityPolicyColumnsUnknown {
		return c, fmt.Errorf("conf read: unknown security policy columns type")
	}

	for _, cf := range c.Security.Defaults.Columns {
		if misc.ValueTypeFromString(cf.Type) == misc.ValueTypeUnknown {
			return c, fmt.Errorf("conf read: unknown default filter type")
		}
	}

	return c, nil
}
