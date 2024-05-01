package gen

import "github.com/dave/jennifer/jen"

// PartialGenerator wraps a jen statement
type PartialGenerator struct {
	raw *jen.Statement
}

// NewPartialGenerator returns a partial generator
func NewPartialGenerator(st *jen.Statement) *PartialGenerator {
	if st != nil {
		return &PartialGenerator{
			raw: st,
		}
	}
	return &PartialGenerator{
		raw: &jen.Statement{},
	}
}
func (p *PartialGenerator) appendMultilineComment(c []string) {
	for i, v := range c {
		if i != len(c)-1 {
			p.raw.Comment(v).Line()
			continue
		}
		p.raw.Comment(v)
	}
}

// Raw returns the jen statement.
func (p *PartialGenerator) Raw() *jen.Statement {
	return p.raw
}

// String returns the source code string
func (p *PartialGenerator) String() string {
	return p.raw.GoString()
}
func (p *PartialGenerator) appendInterface(name string, methods []jen.Code) {
	p.raw.Type().Id(name).Interface(methods...).Line()
}

func (p *PartialGenerator) appendInterfaceType(name string, types []jen.Code) {
	p.raw.Type().Id(name).Interface(jen.Union(types...)).Line()
}

func (p *PartialGenerator) appendStruct(name string, fields ...jen.Code) {
	p.raw.Type().Id(name).Struct(fields...).Line()
}

// NewLine insert a new line in code.
func (p *PartialGenerator) NewLine() {
	p.raw.Line()
}

func (p *PartialGenerator) appendFunction(name string, stp *jen.Statement,
	parameters []jen.Code, results []jen.Code, oneResponse string, body ...jen.Code) {
	p.raw.Func()
	if stp != nil {
		p.raw.Params(stp)
	}
	if name != "" {
		p.raw.Id(name)
	}
	p.raw.Params(parameters...)
	if oneResponse != "" {
		p.raw.Id(oneResponse)
	} else if len(results) > 0 {
		p.raw.Params(results...)
	}
	p.raw.Block(body...)
}

// appendTypeParamFunction appends a `func $name[$paramtypes]($parameters) ($results) { $body }`
func (p *PartialGenerator) appendTypeParamFunction(name string, paramtypes []jen.Code,
	parameters []jen.Code, results []jen.Code, oneResponse string, body ...jen.Code) {
	p.raw.Func()
	if name != "" {
		p.raw.Id(name)
	}
	if paramtypes != nil {
		p.raw.TypesFunc(func(g *jen.Group) {
			g.List(paramtypes...)
		})
	}
	p.raw.Params(parameters...)
	if oneResponse != "" {
		p.raw.Id(oneResponse)
	} else if len(results) > 0 {
		p.raw.Params(results...)
	}

	p.raw.Block(body...)

}

// appendTypeFunction appends a `type $name func[$paramtypes]($params) $returns`
func (p *PartialGenerator) appendTypeFunction(name string, paramtypes []jen.Code,
	params []jen.Code, returns []jen.Code, oneReturn string) {
	p.raw.Type().Id(name)
	if paramtypes != nil {
		p.raw.TypesFunc(func(g *jen.Group) {
			g.List(paramtypes...)
		})
	}
	p.raw.Func()
	p.raw.Params(params...)
	if oneReturn != "" {
		p.raw.Id(oneReturn)
	} else if len(returns) > 0 {
		p.raw.Params(returns...)
	}
}

func (p *PartialGenerator) appendConsts(consts ...jen.Code) {
	p.raw.Const().Defs(consts...).Line()
}

func (p *PartialGenerator) appendVars(vars ...jen.Code) {
	p.raw.Var().Defs(vars...).Line()
}
