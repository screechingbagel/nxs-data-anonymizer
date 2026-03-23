package misc

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	ttemplate "text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/jaswdr/faker/v2"
)

var (
	TemplateNULL = "::NULL::"
	TemplateDrop = "::DROP::"
)

type TemlateRes struct {
	Value   string
	DropRow bool
}

var templateFake = faker.New()

var (
	templateFuncMap     ttemplate.FuncMap
	templateFuncMapOnce sync.Once
	templateInvoiceMu   sync.Mutex
)

func fakerInvoice() string {
	templateInvoiceMu.Lock()
	defer templateInvoiceMu.Unlock()

	counterFile := "/tmp/nxs_invoice_seq"
	count := 1000

	// Try to read the file
	data, err := os.ReadFile(counterFile)
	if err == nil {
		parsed, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil {
			count = parsed
		}
	}

	// Write the incremented count back to the file
	os.WriteFile(counterFile, []byte(strconv.Itoa(count+1)), 0644)
	return fmt.Sprintf("INV-%08d", count)
}

func getTemplateFuncMap() ttemplate.FuncMap {
	templateFuncMapOnce.Do(func() {
		t := sprig.TxtFuncMap()

		t["null"] = func() string { return TemplateNULL }
		t["isNull"] = func(v string) bool { return v == TemplateNULL }
		t["drop"] = func() string { return TemplateDrop }

		// Names
		t["fakerName"] = templateFake.Person().Name
		t["fakerFirstName"] = templateFake.Person().FirstName
		t["fakerLastName"] = templateFake.Person().LastName

		// Contact
		t["fakerEmail"] = templateFake.Internet().Email
		t["fakerPhone"] = templateFake.Phone().Number

		// Address
		t["fakerAddress"] = templateFake.Address().Address
		t["fakerStreetAddress"] = templateFake.Address().StreetAddress
		t["fakerSecondaryAddress"] = templateFake.Address().SecondaryAddress
		t["fakerCity"] = templateFake.Address().City
		t["fakerPostcode"] = templateFake.Address().PostCode

		// Company
		t["fakerCompany"] = templateFake.Company().Name

		// Identifiers
		t["fakerIBAN"] = func() string {
			return strings.ToUpper(templateFake.Bothify("??####################"))
		}
		t["fakerSwift"] = func() string {
			return strings.ToUpper(templateFake.Bothify("??????##"))
		}
		t["fakerEIN"] = func() string {
			return templateFake.Bothify("##-#######")
		}

		t["fakerInvoice"] = fakerInvoice

		templateFuncMap = t
	})
	return templateFuncMap
}

var (
	templateCache   = make(map[string]*ttemplate.Template)
	templateCacheMu sync.RWMutex
)

func GetCompiledTemplate(tpl string) (*ttemplate.Template, error) {
	templateCacheMu.RLock()
	t, ok := templateCache[tpl]
	templateCacheMu.RUnlock()
	if ok {
		return t, nil
	}

	t, err := ttemplate.New("template").Funcs(getTemplateFuncMap()).Parse(tpl)
	if err != nil {
		return nil, err
	}

	templateCacheMu.Lock()
	templateCache[tpl] = t
	templateCacheMu.Unlock()
	return t, nil
}

func TemplateExecTpl(t *ttemplate.Template, d any) (TemlateRes, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, d); err != nil {
		return TemlateRes{}, err
	}

	// Return empty line if buffer is nil
	if b.Bytes() == nil {
		return TemlateRes{
				Value:   "",
				DropRow: false,
			},
			nil
	}

	// Return `drop` value if buffer is DROP (with special key)
	if bytes.Equal(b.Bytes(), []byte(TemplateDrop)) {
		return TemlateRes{
				Value:   "",
				DropRow: true,
			},
			nil
	}

	// Return buffer content otherwise
	return TemlateRes{
			Value:   b.String(),
			DropRow: false,
		},
		nil
}

func TemplateExec(tpl string, d any) (TemlateRes, error) {

	t, err := GetCompiledTemplate(tpl)
	if err != nil {
		return TemlateRes{}, err
	}

	return TemplateExecTpl(t, d)
}
