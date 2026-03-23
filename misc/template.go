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

var (
	invoiceBatchSize = 1000
	invoiceCurrent   = 0
	invoiceMax       = 0
)

func fakerInvoice() string {
	templateInvoiceMu.Lock()
	defer templateInvoiceMu.Unlock()

	counterFile := "/tmp/nxs_invoice_seq"

	// Initialize or refresh batch if needed
	if invoiceCurrent >= invoiceMax {
		count := 1000
		// Try to read the file
		data, err := os.ReadFile(counterFile)
		if err == nil {
			parsed, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				count = parsed
			}
		}

		// Reserve a batch
		invoiceCurrent = count
		invoiceMax = count + invoiceBatchSize

		// Write the next batch start value back to the file
		_ = os.WriteFile(counterFile, []byte(strconv.Itoa(invoiceMax)), 0644)
	}

	// Use next value from batch
	val := invoiceCurrent
	invoiceCurrent++

	return fmt.Sprintf("INV-%08d", val)
}

// Exported wrappers for generated code

func FakerInvoice() string {
	return fakerInvoice()
}

func FakerName() string {
	return templateFake.Person().Name()
}

func FakerFirstName() string {
	return templateFake.Person().FirstName()
}

func FakerLastName() string {
	return templateFake.Person().LastName()
}

func FakerEmail() string {
	return templateFake.Internet().Email()
}

func FakerPhone() string {
	return templateFake.Phone().Number()
}

func FakerAddress() string {
	return templateFake.Address().Address()
}

func FakerStreetAddress() string {
	return templateFake.Address().StreetAddress()
}

func FakerSecondaryAddress() string {
	return templateFake.Address().SecondaryAddress()
}

func FakerCity() string {
	return templateFake.Address().City()
}

func FakerPostcode() string {
	return templateFake.Address().PostCode()
}

func FakerCompany() string {
	return templateFake.Company().Name()
}

func FakerIBAN() string {
	return strings.ToUpper(templateFake.Bothify("??####################"))
}

func FakerSwift() string {
	return strings.ToUpper(templateFake.Bothify("??????##"))
}

func FakerEIN() string {
	return templateFake.Bothify("##-#######")
}

var (
	sprigUuidv4       func() string
	sprigRandAlphaNum func(int) string
	sprigRandNumeric  func(int) string
)

func init() {
	fm := sprig.TxtFuncMap()
	if f, ok := fm["uuidv4"].(func() string); ok {
		sprigUuidv4 = f
	}
	if f, ok := fm["randAlphaNum"].(func(int) string); ok {
		sprigRandAlphaNum = f
	}
	if f, ok := fm["randNumeric"].(func(int) string); ok {
		sprigRandNumeric = f
	}
}

func Uuidv4() string {
	if sprigUuidv4 != nil {
		return sprigUuidv4()
	}
	return ""
}

func RandAlphaNum(n int) string {
	if sprigRandAlphaNum != nil {
		return sprigRandAlphaNum(n)
	}
	return ""
}

func RandNumeric(n int) string {
	if sprigRandNumeric != nil {
		return sprigRandNumeric(n)
	}
	return ""
}

func getTemplateFuncMap() ttemplate.FuncMap {
	templateFuncMapOnce.Do(func() {
		t := sprig.TxtFuncMap()

		t["null"] = func() string { return TemplateNULL }
		t["isNull"] = func(v string) bool { return v == TemplateNULL }
		t["drop"] = func() string { return TemplateDrop }

		// Names
		t["fakerName"] = FakerName
		t["fakerFirstName"] = FakerFirstName
		t["fakerLastName"] = FakerLastName

		// Contact
		t["fakerEmail"] = FakerEmail
		t["fakerPhone"] = FakerPhone

		// Address
		t["fakerAddress"] = FakerAddress
		t["fakerStreetAddress"] = FakerStreetAddress
		t["fakerSecondaryAddress"] = FakerSecondaryAddress
		t["fakerCity"] = FakerCity
		t["fakerPostcode"] = FakerPostcode

		// Company
		t["fakerCompany"] = FakerCompany

		// Identifiers
		t["fakerIBAN"] = FakerIBAN
		t["fakerSwift"] = FakerSwift
		t["fakerEIN"] = FakerEIN

		t["fakerInvoice"] = FakerInvoice

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
