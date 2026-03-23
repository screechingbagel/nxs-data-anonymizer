package mysql_anonymize

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/nixys/nxs-data-anonymizer/misc"
)

func dhSecurityInsertInto(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	uctx.security.tmpBuf = token
	uctx.insertIntoBuf = nil
	uctx.whitespaceBuf = nil

	return deferred, nil
}

func dhSecurityInsertIntoTableNameSearch(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	uctx.security.tmpBuf = append(uctx.security.tmpBuf, deferred...)
	uctx.security.tmpBuf = append(uctx.security.tmpBuf, token...)

	return []byte{}, nil
}

func dhSecurityInsertIntoValues(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	uctx.insertIntoBuf = append(uctx.insertIntoBuf, deferred...)
	uctx.insertIntoBuf = append(uctx.insertIntoBuf, token...)

	return []byte{}, nil
}

func dhSecurityInsertIntoValueSearch(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	if uctx.insertIntoBuf != nil {
		uctx.insertIntoBuf = append(uctx.insertIntoBuf, deferred...)
	} else {
		uctx.whitespaceBuf = append(uctx.whitespaceBuf, deferred...)
	}

	return []byte{}, nil
}

func dhSecurityValuesEnd(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	if uctx.insertIntoBuf != nil {
		return []byte{}, nil
	}

	return append(deferred, token...), nil
}

func dhCreateTableName(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)
	uctx.filter.TableCreate(string(deferred))

	return append(deferred, token...), nil
}

func dhCreateTableFieldName(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)
	uctx.columnName = string(deferred)

	return append(deferred, token...), nil
}

// generatedRgx detects generated/virtual columns.
// See: https://dev.mysql.com/blog-archive/generated-columns-in-mysql-5-7-5
var generatedRgx = regexp.MustCompile(`^([A-Z]+)(\([0-9]+\) | )(GENERATED ALWAYS AS|AS)`)

var asBytes = []byte("AS")

func checkGenerated(t []byte) bool {
	if bytes.Contains(t, asBytes) {
		return generatedRgx.Match(t)
	}
	return false
}

func dhCreateTableColumnAdd(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	traw := bytes.TrimSpace(deferred)
	trawUpper := bytes.ToUpper(traw)

	if !checkGenerated(trawUpper) {

		t, b := uctx.tables[uctx.filter.TableNameGet()]
		if !b {
			t = make(map[string]columnType)
		}
		t[uctx.columnName] = columnTypeNone

		for _, pk := range typePrefixes {
			if bytes.HasPrefix(trawUpper, pk.p) {
				t[uctx.columnName] = pk.t
				break
			}
		}

		uctx.tables[uctx.filter.TableNameGet()] = t
		uctx.filter.ColumnAdd(uctx.columnName, string(traw))
	}

	uctx.columnName = ""

	return append(deferred, token...), nil
}

func dhInsertIntoTableName(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	tn := string(deferred)

	// Check table pass through security rules
	if !securityPolicyCheck(uctx, tn) {

		// If not: table will be skipped from result dump

		uctx.security.isSkip = true
		uctx.security.tmpBuf = []byte{}
		return []byte{}, nil
	}

	uctx.insertIntoBuf = append(uctx.security.tmpBuf, deferred...)
	uctx.insertIntoBuf = append(uctx.insertIntoBuf, token...)

	uctx.security.isSkip = false
	uctx.security.tmpBuf = []byte{}

	// Check insert into table name
	if tn != uctx.filter.TableNameGet() {
		return []byte{}, fmt.Errorf("`create` and `insert into` table names are mismatch (create table: '%s', insert into table: '%s')", uctx.filter.TableNameGet(), tn)
	}

	return []byte{}, nil
}

func dhCreateTableValues(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	s := string(deferred)
	if s == "NULL" {
		uctx.filter.ValueAdd(misc.TemplateNULL)
	} else {
		uctx.filter.ValueAdd(s)
	}

	return []byte{}, nil
}

func dhCreateTableValuesString(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	s := string(deferred)
	uctx.filter.ValueAdd(s)

	return []byte{}, nil
}

func dhCreateTableValuesEnd(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	s := string(deferred)
	if s == "NULL" {
		uctx.filter.ValueAdd(misc.TemplateNULL)
	} else {
		uctx.filter.ValueAdd(s)
	}

	// Apply filter for row
	if err := uctx.filter.Apply(); err != nil {
		return []byte{}, err
	}

	b := rowDataGen(uctx)
	if b == nil {
		return []byte{}, nil
	} else {
		if uctx.insertIntoBuf != nil {
			b = append(uctx.insertIntoBuf, b...)
			uctx.insertIntoBuf = nil
		} else {
			if uctx.whitespaceBuf != nil {
				b = append(uctx.whitespaceBuf, b...)
				uctx.whitespaceBuf = nil
			}
			b = append([]byte{','}, b...)
		}
	}

	return b, nil
}

func dhCreateTableValuesStringEnd(usrCtx any, deferred, token []byte) ([]byte, error) {

	uctx := usrCtx.(*userCtx)

	if uctx.security.isSkip {
		return []byte{}, nil
	}

	// Apply filter for row
	if err := uctx.filter.Apply(); err != nil {
		return []byte{}, err
	}

	b := rowDataGen(uctx)
	if b == nil {
		return []byte{}, nil
	} else {
		if uctx.insertIntoBuf != nil {
			b = append(uctx.insertIntoBuf, b...)
			uctx.insertIntoBuf = nil
		} else {
			if uctx.whitespaceBuf != nil {
				b = append(uctx.whitespaceBuf, b...)
				uctx.whitespaceBuf = nil
			}
			b = append([]byte{','}, b...)
		}
	}

	return b, nil
}

func rowDataGen(uctx *userCtx) []byte {

	row := uctx.filter.ValuePop()
	if row.Values == nil {
		return nil
	}

	var out []byte
	out = append(out, '(')

	tname := uctx.filter.TableNameGet()
	tableCols := uctx.tables[tname]

	for i, v := range row.Values {

		if i > 0 {
			out = append(out, ',')
		}

		if v.V == misc.TemplateNULL {
			out = append(out, "NULL"...)
		} else {
			cname := uctx.filter.ColumnGetName(i)
			switch tableCols[cname] {
			case columnTypeString:
				out = append(out, '\'')
				out = append(out, v.V...)
				out = append(out, '\'')
			case columnTypeBinary:
				out = append(out, "_binary '"...)
				out = append(out, v.V...)
				out = append(out, '\'')
			default:
				out = append(out, v.V...)
			}
		}
	}

	out = append(out, ')')
	return out
}

// SecurityPolicyCheck checks the table passes the security rules
// true:  pass
// false: skip
func securityPolicyCheck(uctx *userCtx, tname string) bool {

	// Continue if security policy is `skip`
	if uctx.security.tablesPolicy != misc.SecurityPolicyTablesSkip {
		return true
	}

	// Check rules for specified table name
	if tr := uctx.filter.TableRulesLookup(tname); tr != nil {
		return true
	}

	// Check specified table name in exceptions
	if _, b := uctx.security.tableExceptions[tname]; b {
		return true
	}

	return false
}
