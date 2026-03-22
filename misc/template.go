package misc

import (
	"bytes"
	"strings"
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

// TemplateExec makes message from given template `tpl` and data `d`
func TemplateExec(tpl string, d any) (TemlateRes, error) {

	var b bytes.Buffer

	// faker integration for realistic data
	fake := faker.New()

	// See http://masterminds.github.io/sprig/ for details
	t, err := ttemplate.New("template").Funcs(func() ttemplate.FuncMap {

		// Get current sprig functions
		t := sprig.TxtFuncMap()

		// Add additional functions
		t["null"] = func() string {
			return TemplateNULL
		}
		t["isNull"] = func(v string) bool {
			return v == TemplateNULL
		}
		t["drop"] = func() string {
			return TemplateDrop
		}

		// Names
		t["fakerName"] = fake.Person().Name
		t["fakerFirstName"] = fake.Person().FirstName
		t["fakerLastName"] = fake.Person().LastName

		// Contact
		t["fakerEmail"] = fake.Internet().Email
		t["fakerPhone"] = fake.Phone().Number

		// Address
		t["fakerAddress"] = fake.Address().Address
		t["fakerStreetAddress"] = fake.Address().StreetAddress
		t["fakerSecondaryAddress"] = fake.Address().SecondaryAddress
		t["fakerCity"] = fake.Address().City
		t["fakerPostcode"] = fake.Address().PostCode

		// Company
		t["fakerCompany"] = fake.Company().Name

		// Identifiers
		t["fakerIBAN"] = func() string {
			return strings.ToUpper(fake.Bothify("??####################"))
		}
		t["fakerSwift"] = func() string {
			return strings.ToUpper(fake.Bothify("??????##"))
		}
		t["fakerEIN"] = func() string {
			return fake.Bothify("##-#######")
		}

		return t
	}()).Parse(tpl)
	if err != nil {
		return TemlateRes{}, err
	}

	err = t.Execute(&b, d)
	if err != nil {
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
