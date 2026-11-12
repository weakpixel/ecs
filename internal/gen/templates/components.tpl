{{define "components"}}
{{if .HasComponents}}
// Components{{.N}} holds pointers to {{.N}} components, in type-parameter
// order. It is only returned by value; its fields are only valid as long as
// the pointers themselves are (see ViewN.Get and ViewN.Each).
type Components{{.N}}[{{.TypeParamDecl}}] struct {
{{range .Fields}}	{{.Comp}} *{{.Name}}
{{end}}}

// Unpack returns the pointers as separate values.
func (c Components{{.N}}[{{.TypeParams}}]) Unpack() ({{.UnpackReturn}}) {
	return {{.UnpackBody}}
}
{{end}}
{{end}}
