package srcobj

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/sirkon/gosrcfmt"
)

// File represents LDE generated Go source file
type File struct {
	pkgName   string
	imports   map[string]string
	strConsts map[string]string

	body *Body
}

// NewFile constructor
func NewFile() *File {
	return &File{
		imports:   map[string]string{},
		strConsts: map[string]string{},
		body:      &Body{},
	}
}

// PkgName sets package name
func (f *File) PkgName(name string) {
	f.pkgName = name
}

// AddConst adds text constant and returns its name
func (f *File) AddConst(name, value string) string {
	suffix := ""
	i := 1
	for {
		candidate := name + suffix
		if val, ok := f.strConsts[candidate]; !ok {
			f.strConsts[candidate] = value
			return candidate
		} else if val == value {
			return candidate

		}
		i++
		suffix = fmt.Sprint(i)
	}
}

// AddNamedImport adds new import item with specific access name
func (f *File) AddNamedImport(access, path string) error {
	if prevAccess, ok := f.imports[path]; ok && prevAccess != access {
		return fmt.Errorf(
			`attempt to use "%s" as '%s' while it was added with '%s' access name before"`,
			path, access, prevAccess)
	}
	f.imports[path] = access
	return nil
}

// AddExtractor adds new extractor struct type definition and returns struct body
func (f *File) AddExtractor(typeName string) *Struct {
	st := structType{
		name: typeName,
		s:    &Struct{},
	}
	f.body.Append(st)
	return st.s
}

// AddExtract adds extraction method for an extractor
func (f *File) AddExtract(typeName string) *Method {
	res := NewExtractor(typeName)
	f.body.Append(Raw("\n"))
	f.body.Append(res)
	return res
}

// AddAccessor adds accessor method for an extractor
func (f *File) AddAccessor(typeName, name string, resultType hardToAccessResultType) *Method {
	res := NewAccessor(typeName, name, resultType)
	f.body.Append(res)
	return res
}

// Dump ...
func (f *File) Dump(w io.Writer) error {
	if len(f.pkgName) == 0 {
		return fmt.Errorf("package name is not set, use PkgName")
	}

	buf := &bytes.Buffer{}
	if _, err := fmt.Fprintf(buf, "package %s\n", f.pkgName); err != nil {
		return err
	}
	if _, err := io.WriteString(buf, "import (\n"); err != nil {
		return err
	}

	imports := []string{}
	for k := range f.imports {
		imports = append(imports, k)
	}
	sort.Sort(sort.StringSlice(imports))
	for _, k := range imports {
		access := f.imports[k]
		if _, err := fmt.Fprintf(buf, `%s "%s"`, access, k); err != nil {
			return err
		}
		if _, err := io.WriteString(buf, "\n"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(buf, ")\n"); err != nil {
		return err
	}

	vars := []string{}
	for n := range f.strConsts {
		vars = append(vars, n)
	}
	sort.Sort(sort.StringSlice(vars))
	for _, varName := range vars {
		value := f.strConsts[varName]
		if _, err := fmt.Fprintf(buf, "var %s = []byte(%s)\n", varName, value); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}

	if err := f.body.Dump(buf); err != nil {
		return err
	}

	return func() (err error) {
		defer func() {
			var ok bool
			if r := recover(); r != nil {
				err, ok = r.(error)
				if !ok {
					err = fmt.Errorf("%s", r)
				}
			}
		}()

		gosrcfmt.Format(w, buf.Bytes())
		return
	}()
}

// Append appends to file body
func (f *File) Append(src Source) {
	f.body.Append(src)
}
