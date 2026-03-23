package relfilter

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	ttemplate "text/template"

	"github.com/nixys/nxs-data-anonymizer/misc"
)

type InitOpts struct {
	Variables map[string]VariableRuleOpts

	Link []LinkOpts

	TableRules       map[string]map[string]ColumnRuleOpts
	DefaultRules     map[string]ColumnRuleOpts
	ExceptionColumns []string

	ColumnsPolicy misc.SecurityPolicyColumnsType

	TypeRuleCustom  []TypeRuleOpts
	TypeRuleDefault []TypeRuleOpts
}

type TypeRuleOpts struct {
	Selector string
	Rule     ColumnRuleOpts
}

type ColumnRuleOpts struct {
	Type   misc.ValueType
	Value  string
	Unique bool
}

type VariableRuleOpts struct {
	Type  misc.ValueType
	Value string
}

type LinkOpts struct {
	Rule ColumnRuleOpts
	With map[string][]string
}

type Filter struct {

	// Rules for filter a table values
	rules rules

	// Temp table data for filtering
	tableData tableData
}

type Row struct {
	Values []rowValue
}

type rules struct {
	variables map[string]string

	tableRules       map[string]map[string]ColumnRuleOpts
	defaultRules     map[string]ColumnRuleOpts
	exceptionColumns map[string]any

	columnsPolicy misc.SecurityPolicyColumnsType

	typeRuleCustom  []typeRule
	typeRuleDefault []typeRule

	link []linkValues

	valEnvGlob []string
}

type typeRule struct {
	Rgx  *regexp.Regexp
	Rule ColumnRuleOpts
}

type tableData struct {
	name    string
	columns columns
	values  []rowValue
	uniques map[string]map[string]any

	// Reusable scratch space for applyRules — allocated once per table and
	// reset (not reallocated) on every row to reduce GC pressure.
	valOld    map[string]string
	valEnvOld []string

	rules      []applyRule
	rulesReady bool

	// Buffers for GlobalRowProcessor optimization
	cnames []string
	vbuf   []string
}

type rowValue struct {
	V string
}

type linkValues struct {

	// Linked tables and columns
	t map[string]map[string]any

	// Map old:new values
	v map[string]string

	// Unique map
	u map[string]any

	// Rule
	r ColumnRuleOpts
}

type execFilterOpts struct {
	t   misc.ValueType
	v   string
	tpl *ttemplate.Template
}

const uniqueAttempts = 5

const (
	envVarGlobalPrefix          = "ENVVARGLOBAL_"
	envVarTable                 = "ENVVARTABLE"
	envVarColumnPrefix          = "ENVVARCOLUMN_"
	envVarCurColumn             = "ENVVARCURCOLUMN"
	envVarColumnTypeRAW         = "ENVVARCOLUMNTYPERAW"
	envVarColumnTypeGroupPrefix = "ENVVARCOLUMNTYPEGROUP_"
)

type applyRule struct {
	c   *column
	i   int
	cr  ColumnRuleOpts
	lv  map[string]string
	u   map[string]any
	tpl *ttemplate.Template
}

func newApplyRule(c *column, i int, cr ColumnRuleOpts, lv map[string]string, u map[string]any) (applyRule, error) {
	rl := applyRule{
		c:  c,
		i:  i,
		cr: cr,
		lv: lv,
		u:  u,
	}
	if cr.Type == misc.ValueTypeTemplate {
		tpl, err := misc.GetCompiledTemplate(cr.Value)
		if err != nil {
			return rl, err
		}
		rl.tpl = tpl
	}
	return rl, nil
}

func Init(opts InitOpts) (*Filter, error) {

	trc := []typeRule{}
	trd := []typeRule{}

	// Make custom type rules
	for _, r := range opts.TypeRuleCustom {

		re, err := regexp.Compile(r.Selector)
		if err != nil {
			return nil, fmt.Errorf("filter init: %w", err)
		}

		trc = append(
			trc,
			typeRule{
				Rgx:  re,
				Rule: r.Rule,
			},
		)
	}

	// Make default type rules
	for _, r := range opts.TypeRuleDefault {

		re, err := regexp.Compile(r.Selector)
		if err != nil {
			return nil, fmt.Errorf("filter init: %w", err)
		}

		trd = append(
			trd,
			typeRule{
				Rgx:  re,
				Rule: r.Rule,
			},
		)
	}

	// Make exceptions
	excpts := make(map[string]any)
	for _, e := range opts.ExceptionColumns {
		excpts[e] = nil
	}

	vars := make(map[string]string)
	for n, f := range opts.Variables {
		v, _, err := execFilter(
			execFilterOpts{
				t: f.Type,
				v: f.Value,
			},
			nil,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("filter init: %w", err)
		}
		vars[n] = v
	}

	// Pre-compute global env vars once;
	valEnvGlob := make([]string, 0, len(vars))
	for n, v := range vars {
		valEnvGlob = append(valEnvGlob, fmt.Sprintf("%s%s=%s", envVarGlobalPrefix, n, v))
	}

	lvs := []linkValues{}
	// Make links
	for _, l := range opts.Link {

		lv := linkValues{
			t: make(map[string]map[string]any),
			v: make(map[string]string),
			u: func() map[string]any {
				if l.Rule.Unique {
					return make(map[string]any)
				}
				return nil
			}(),
			r: l.Rule,
		}

		for t, cs := range l.With {
			m := make(map[string]any)
			for _, c := range cs {
				m[c] = nil
			}
			lv.t[t] = m
		}

		lvs = append(lvs, lv)
	}

	return &Filter{
		rules: rules{
			variables:        vars,
			valEnvGlob:       valEnvGlob,
			link:             lvs,
			tableRules:       opts.TableRules,
			defaultRules:     opts.DefaultRules,
			exceptionColumns: excpts,
			typeRuleCustom:   trc,
			typeRuleDefault:  trd,
			columnsPolicy:    opts.ColumnsPolicy,
		},
	}, nil
}

// TableCreate creates new data set for table `name`
func (filter *Filter) TableCreate(name string) {
	filter.tableData = tableData{
		name:       name,
		columns:    columnsInit(),
		uniques:    make(map[string]map[string]any),
		values:     []rowValue{},
		valOld:     make(map[string]string),
		valEnvOld:  []string{},
		rules:      []applyRule{},
		rulesReady: false,
	}
}

func (filter *Filter) TableNameGet() string {
	return filter.tableData.name
}

// TableRulesLookup looks up filters for specified table name
func (filter *Filter) TableRulesLookup(name string) map[string]ColumnRuleOpts {
	if t, b := filter.rules.tableRules[name]; b {
		return t
	}
	return nil
}

// ColumnAdd adds new column into current data set
func (filter *Filter) ColumnAdd(name string, rt string) {

	//var rl *ColumnRuleOpts

	for _, r := range filter.rules.typeRuleCustom {
		gd := r.Rgx.FindAllStringSubmatch(rt, -1)
		if len(gd) > 0 {
			filter.tableData.columns.add(name, rt, gd, &r.Rule)
			return
		}
	}

	for _, r := range filter.rules.typeRuleDefault {
		gd := r.Rgx.FindAllStringSubmatch(rt, -1)
		if len(gd) > 0 {
			filter.tableData.columns.add(name, rt, gd, &r.Rule)
			return
		}
	}

	filter.tableData.columns.add(name, rt, nil, nil)
}

func (filter *Filter) ColumnGetName(index int) string {
	return filter.tableData.columns.getNameByIndex(index)
}

func (filter *Filter) ValueAdd(b string) {
	filter.tableData.values = append(
		filter.tableData.values,
		rowValue{
			V: b,
		},
	)
}

// ValuePop pops the last values row from current data set
func (filter *Filter) ValuePop() Row {

	// Save current values
	r := filter.tableData.values

	filter.rowCleanup()

	return Row{
		Values: r,
	}
}

var GlobalRowProcessor func(tableName string, columns []string, values []string) ([]string, error)

func (filter *Filter) Apply() error {

	if filter.tableData.rulesReady {
		if err := filter.applyRules(filter.tableData.name, filter.tableData.rules); err != nil {
			return fmt.Errorf("filters apply: %w", err)
		}
		return nil
	}

	// Fast path for generated code
	if GlobalRowProcessor != nil {

		// Initialize buffers if needed
		if filter.tableData.cnames == nil {
			count := len(filter.tableData.columns.cc)
			filter.tableData.cnames = make([]string, count)
			filter.tableData.vbuf = make([]string, count)
			for i, c := range filter.tableData.columns.cc {
				filter.tableData.cnames[i] = c.n
			}
		}

		// Copy values to vbuf
		for i, v := range filter.tableData.values {
			filter.tableData.vbuf[i] = v.V
		}

		newValues, err := GlobalRowProcessor(filter.tableData.name, filter.tableData.cnames, filter.tableData.vbuf)
		if err != nil {
			return err
		}

		// Update values
		for i, v := range newValues {
			filter.tableData.values[i].V = v
		}
		return nil
	}

	var rls []applyRule

	tname := filter.tableData.name

	// Check rules exist for current table
	tr := filter.TableRulesLookup(tname)

	// Create rules for every column within current table
	for i, c := range filter.tableData.columns.cc {

		// Check linked column
		t := false
		for _, l := range filter.rules.link {
			if e, b := l.t[tname]; b {
				if _, u := e[c.n]; u {
					rl, err := newApplyRule(c, i, l.r, l.v, l.u)
					if err != nil {
						return fmt.Errorf("filters apply link: %w", err)
					}
					rls = append(rls, rl)
					t = true
					break
				}
			}
		}
		if t {
			continue
		}

		// Check direct rules for column
		if tr != nil {
			if cr, e := tr[c.n]; e {
				rl, err := newApplyRule(c, i, cr, nil, nil)
				if err != nil {
					return fmt.Errorf("filters apply direct: %w", err)
				}
				rls = append(rls, rl)
				continue
			}
		}

		// Check default rules for column
		if cr, e := filter.rules.defaultRules[c.n]; e {
			rl, err := newApplyRule(c, i, cr, nil, nil)
			if err != nil {
				return fmt.Errorf("filters apply default: %w", err)
			}
			rls = append(rls, rl)
			continue
		}

		// Check column is excepted
		if _, b := filter.rules.exceptionColumns[c.n]; b {
			continue
		}

		// Other rules if required

		// Default rules for types
		if filter.rules.columnsPolicy == misc.SecurityPolicyColumnsRandomize {
			if c.t.r != nil {
				rl, err := newApplyRule(c, i, *c.t.r, nil, nil)
				if err != nil {
					return fmt.Errorf("filters apply randomize: %w", err)
				}
				rls = append(rls, rl)
			}
		}
	}

	// Apply rules
	filter.tableData.rules = rls
	filter.tableData.rulesReady = true

	if err := filter.applyRules(tname, filter.tableData.rules); err != nil {
		return fmt.Errorf("filters apply: %w", err)
	}

	return nil
}

func (filter *Filter) applyRules(tname string, rls []applyRule) error {

	// If no columns has rules
	if len(rls) == 0 {
		return nil
	}

	valEnvGlob := filter.rules.valEnvGlob

	hasCmd := false
	for _, r := range rls {
		if r.cr.Type == misc.ValueTypeCommand {
			hasCmd = true
			break
		}
	}

	// Reuse the per-table maps and slices
	valOld := filter.tableData.valOld
	valEnvOld := filter.tableData.valEnvOld[:0]

	for i, c := range filter.tableData.columns.cc {
		valOld[c.n] = filter.tableData.values[i].V
		if hasCmd {
			valEnvOld = append(
				valEnvOld,
				fmt.Sprintf("%s%s=%s", envVarColumnPrefix, c.n, filter.tableData.values[i].V),
			)
		}
	}
	if hasCmd {
		filter.tableData.valEnvOld = valEnvOld
	}

	// Apply rule for each specified column
	for _, r := range rls {

		var (
			v   string
			d   bool
			err error
		)

		if r.lv != nil {

			// For linked columns

			td := struct {
				Variables map[string]string
			}{
				Variables: filter.rules.variables,
			}

			if vo := valOld[r.c.n]; vo == misc.TemplateNULL {

				// If old value for this cell is NULL

				v, d, err = filter.applyLinkFilter(r.c.n, r.cr, r.tpl, r.u, td, valEnvGlob)
				if err != nil {
					return fmt.Errorf("rules: %w", err)
				}

				if d {
					filter.tableData.values = nil
					return nil
				}
			} else {

				// Check linked value for this column already exist
				if e, b := r.lv[vo]; b {
					v = e
				} else {
					v, d, err = filter.applyLinkFilter(r.c.n, r.cr, r.tpl, r.u, td, valEnvGlob)
					if err != nil {
						return fmt.Errorf("rules: %w", err)
					}

					if d {
						filter.tableData.values = nil
						return nil
					}

					r.lv[vo] = v
				}
			}
		} else {

			type tplData struct {
				TableName        string
				CurColumnName    string
				Values           map[string]string
				Variables        map[string]string
				ColumnTypeRaw    string
				ColumnTypeGroups [][]string
			}

			td := tplData{
				TableName:        tname,
				CurColumnName:    r.c.n,
				Values:           valOld,
				Variables:        filter.rules.variables,
				ColumnTypeRaw:    r.c.t.raw,
				ColumnTypeGroups: r.c.t.groups,
			}

			var tde []string
			if r.cr.Type == misc.ValueTypeCommand {
				tde = []string{
					fmt.Sprintf("%s=%s", envVarTable, tname),
					fmt.Sprintf("%s=%s", envVarCurColumn, r.c.n),
				}
				tde = append(tde, valEnvOld...)
				tde = append(tde, valEnvGlob...)
				tde = append(tde, r.c.t.env...)
			}

			v, d, err = filter.applyColumnFilter(r.c.n, r.cr, r.tpl, td, tde)
			if err != nil {
				return fmt.Errorf("rules: %w", err)
			}

			if d {
				filter.tableData.values = nil
				return nil
			}
		}

		// Set specified value in accordance with filter
		filter.tableData.values[r.i].V = v
	}

	return nil
}

func (filter *Filter) applyColumnFilter(cn string, cr ColumnRuleOpts, tpl *ttemplate.Template, td any, tde []string) (string, bool, error) {

	for range uniqueAttempts {

		v, d, err := execFilter(
			execFilterOpts{
				t:   cr.Type,
				v:   cr.Value,
				tpl: tpl,
			},
			td,
			tde)
		if err != nil {
			return "", false, fmt.Errorf("apply filter: %w", err)
		}

		if d {
			return "", true, nil
		}

		if v == misc.TemplateNULL {
			return v, false, nil
		}

		if !cr.Unique {
			return v, false, nil
		}

		var uv map[string]any
		if _, b := filter.tableData.uniques[cn]; !b {
			// For first values
			uv = make(map[string]any)
		} else {
			uv = filter.tableData.uniques[cn]
		}

		if _, b := uv[v]; !b {
			uv[v] = nil
			filter.tableData.uniques[cn] = uv
			return v, false, nil
		}
	}

	return "", false, fmt.Errorf("filter: unable to generate unique value for column `%s.%s`, check filter value for this column in config", filter.tableData.name, cn)
}

func (filter *Filter) applyLinkFilter(cn string, cr ColumnRuleOpts, tpl *ttemplate.Template, u map[string]any, td any, tde []string) (string, bool, error) {

	for range uniqueAttempts {

		v, d, err := execFilter(
			execFilterOpts{
				t:   cr.Type,
				v:   cr.Value,
				tpl: tpl,
			},
			td,
			tde)
		if err != nil {
			return "", false, fmt.Errorf("apply link filter: %w", err)
		}

		if d {
			return "", true, nil
		}

		if v == misc.TemplateNULL {
			return v, false, nil
		}

		if !cr.Unique {
			return v, false, nil
		}

		if _, b := u[v]; !b {
			u[v] = nil
			return v, false, nil
		}
	}

	return "", false, fmt.Errorf("apply link filter: unable to generate unique value for column `%s.%s`, check filter value for this column in config", filter.tableData.name, cn)
}

// rowCleanup cleanups current row values
func (filter *Filter) rowCleanup() {
	filter.tableData.values = []rowValue{}
}

func execFilter(f execFilterOpts, td any, tde []string) (string, bool, error) {

	var (
		r   string
		d   bool
		err error
	)

	switch f.t {
	case misc.ValueTypeTemplate:
		var t misc.TemlateRes
		if f.tpl != nil {
			t, err = misc.TemplateExecTpl(f.tpl, td)
		} else {
			t, err = misc.TemplateExec(f.v, td) // fallback
		}
		if err != nil {
			return "", false, fmt.Errorf("filter: value compile template: %w", err)
		}

		r = t.Value
		d = t.DropRow
	case misc.ValueTypeCommand:

		var stderr, stdout bytes.Buffer

		parsed_cmd := strings.Split(f.v, " ")
		name := parsed_cmd[0]
		args := parsed_cmd[1:]
		cmd := exec.Command(name, args...)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		cmd.Env = tde

		if err := cmd.Run(); err != nil {

			e, b := err.(*exec.ExitError)
			if !b {
				return "", false, fmt.Errorf("filter: value exec command: %w", err)
			}

			return "", false, fmt.Errorf("filter: value exec command: bad exit code %d: %s", e.ExitCode(), stderr.String())
		}

		r = stdout.String()

	default:
		return "", false, fmt.Errorf("filter: value compile: unknown type")
	}

	return strings.ReplaceAll(r, "\n", "\\n"), d, nil
}
