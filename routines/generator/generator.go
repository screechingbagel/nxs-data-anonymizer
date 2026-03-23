package generator

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/nixys/nxs-data-anonymizer/ctx"
)

type Generator struct {
	imports map[string]bool
}

func NewGenerator() *Generator {
	return &Generator{
		imports: make(map[string]bool),
	}
}

func (g *Generator) addImport(pkg string) {
	g.imports[pkg] = true
}

func Run(confPath string, outputPath string) error {
	conf, err := ctx.ConfRead(confPath)
	if err != nil {
		return err
	}

	gen := NewGenerator()
	var body bytes.Buffer

	// We always need these for the hook registration
	gen.addImport("github.com/nixys/nxs-data-anonymizer/modules/filters/relfilter")

	// Start generating the processor function body
	body.WriteString("func fastRowProcessor(tableName string, columns []string, values []string) ([]string, error) {\n")
	body.WriteString("\tswitch tableName {\n")

	// Iterate tables
	for tName, tConf := range conf.Filters {
		fmt.Fprintf(&body, "\tcase \"%s\":\n", tName)
		body.WriteString("\t\tfor i, col := range columns {\n")
		body.WriteString("\t\t\tswitch col {\n")

		for cName, cConf := range tConf.Columns {
			code := gen.generateRuleCode(cConf.Value)
			fmt.Fprintf(&body, "\t\t\tcase \"%s\":\n", cName)
			fmt.Fprintf(&body, "\t\t\t\tvalues[i] = %s\n", code)
		}
		body.WriteString("\t\t\t}\n") // end switch col
		body.WriteString("\t\t}\n")   // end for
	}

	body.WriteString("\t}\n") // end switch table
	body.WriteString("\treturn values, nil\n")
	body.WriteString("}\n")

	// Now combine everything into the final file
	var fileContent bytes.Buffer
	fileContent.WriteString("package main\n\n")
	fileContent.WriteString("import (\n")

	// Sort imports for stability
	sortedImports := make([]string, 0, len(gen.imports))
	for pkg := range gen.imports {
		sortedImports = append(sortedImports, pkg)
	}
	sort.Strings(sortedImports)

	for _, pkg := range sortedImports {
		fmt.Fprintf(&fileContent, "\t\"%s\"\n", pkg)
	}
	fileContent.WriteString(")\n\n")

	// Add the init hook
	fileContent.WriteString(`func init() {
	relfilter.GlobalRowProcessor = fastRowProcessor
}

`)

	// Add the body
	fileContent.Write(body.Bytes())

	return os.WriteFile(outputPath, fileContent.Bytes(), 0644)
}

func (g *Generator) generateRuleCode(val string) string {
	val = strings.TrimSpace(val)
	re := regexp.MustCompile(`\{\{(.*?)\}\}`)

	parts := re.Split(val, -1)
	matches := re.FindAllStringSubmatch(val, -1)

	var codeParts []string

	for i, part := range parts {
		if part != "" {
			if strings.Contains(part, "\"") {
				// If the literal contains double quotes, use fmt.Sprintf or raw string?
				// Simplest is standard %q formatting which handles escaping.
				g.addImport("fmt")
				codeParts = append(codeParts, fmt.Sprintf("fmt.Sprintf(\"%%s\", %q)", part))
			} else {
				codeParts = append(codeParts, fmt.Sprintf("%q", part))
			}
		}
		if i < len(matches) {
			generated := g.generateTemplateExpression(matches[i][1])
			codeParts = append(codeParts, generated)
		}
	}

	if len(codeParts) == 0 {
		return "\"\""
	}

	// Join with +
	// Ensure `strings` is imported if we used Join? No, we use + operator in Go code.
	// But `strings.Join` in generator is for creating the string of code.
	return strings.Join(codeParts, " + ")
}

func (g *Generator) generateTemplateExpression(content string) string {
	content = strings.TrimSpace(content)
	parts := strings.Split(content, "|")
	funcCall := strings.TrimSpace(parts[0])

	var code string

	re := regexp.MustCompile(`^(\w+)(?:\s+(.*))?$`)
	matches := re.FindStringSubmatch(funcCall)

	if len(matches) > 0 {
		fName := matches[1]
		args := strings.TrimSpace(matches[2])

		// For any misc.* function, we need the import
		// We add it lazily but it's safe to add multiple times (map keys)
		g.addImport("github.com/nixys/nxs-data-anonymizer/misc")

		switch fName {
		case "null":
			code = "misc.TemplateNULL"
		case "fakerName":
			code = "misc.FakerName()"
		case "fakerFirstName":
			code = "misc.FakerFirstName()"
		case "fakerLastName":
			code = "misc.FakerLastName()"
		case "fakerEmail":
			code = "misc.FakerEmail()"
		case "fakerPhone":
			code = "misc.FakerPhone()"
		case "fakerCompany":
			code = "misc.FakerCompany()"
		case "fakerAddress":
			code = "misc.FakerAddress()"
		case "fakerStreetAddress":
			code = "misc.FakerStreetAddress()"
		case "fakerSecondaryAddress":
			code = "misc.FakerSecondaryAddress()"
		case "fakerCity":
			code = "misc.FakerCity()"
		case "fakerPostcode":
			code = "misc.FakerPostcode()"
		case "fakerIBAN":
			code = "misc.FakerIBAN()"
		case "fakerSwift":
			code = "misc.FakerSwift()"
		case "fakerEIN":
			code = "misc.FakerEIN()"
		case "fakerInvoice":
			code = "misc.FakerInvoice()"
		case "uuidv4":
			code = "misc.Uuidv4()"
		case "randAlphaNum":
			code = fmt.Sprintf("misc.RandAlphaNum(%s)", args)
		case "randNumeric":
			code = fmt.Sprintf("misc.RandNumeric(%s)", args)
		default:
			// Fallback needs fmt
			g.addImport("fmt")
			return fmt.Sprintf("fmt.Sprintf(\"{{%%s}}\", %q)", content)
		}
	} else {
		// Just literal inside braces?
		if content == "null" {
			g.addImport("github.com/nixys/nxs-data-anonymizer/misc")
			code = "misc.TemplateNULL"
		} else {
			g.addImport("fmt")
			return fmt.Sprintf("fmt.Sprintf(\"{{%%s}}\", %q)", content)
		}
	}

	for i := 1; i < len(parts); i++ {
		pipe := strings.TrimSpace(parts[i])
		if pipe == "upper" {
			g.addImport("strings")
			code = fmt.Sprintf("strings.ToUpper(%s)", code)
		}
	}

	return code
}
