package printer

import (
	"strconv"
	"strings"

	"github.com/mmcloughlin/avo"
	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/operand"
)

// dot is the pesky unicode dot used in Go assembly.
const dot = "\u00b7"

type goasm struct {
	cfg Config
	prnt.Generator
}

func NewGoAsm(cfg Config) Printer {
	return &goasm{cfg: cfg}
}

func (p *goasm) Print(f *avo.File) ([]byte, error) {
	p.header()
	p.includes(f.Includes)
	for _, s := range f.Sections {
		switch s := s.(type) {
		case *avo.Function:
			p.function(s)
		case *avo.Global:
			p.global(s)
		default:
			panic("unknown section type")
		}
	}
	return p.Result()
}

func (p *goasm) header() {
	p.Comment(p.cfg.GeneratedWarning())
}

func (p *goasm) includes(paths []string) {
	if len(paths) == 0 {
		return
	}
	p.NL()
	for _, path := range paths {
		p.Printf("#include \"%s\"\n", path)
	}
}

func (p *goasm) function(f *avo.Function) {
	p.NL()
	p.Comment(f.Stub())

	// Reference: https://github.com/golang/go/blob/b115207baf6c2decc3820ada4574ef4e5ad940ec/src/cmd/internal/obj/util.go#L166-L176
	//
	//		if p.As == ATEXT {
	//			// If there are attributes, print them. Otherwise, skip the comma.
	//			// In short, print one of these two:
	//			// TEXT	foo(SB), DUPOK|NOSPLIT, $0
	//			// TEXT	foo(SB), $0
	//			s := p.From.Sym.Attribute.TextAttrString()
	//			if s != "" {
	//				fmt.Fprintf(&buf, "%s%s", sep, s)
	//				sep = ", "
	//			}
	//		}
	//
	p.Printf("TEXT %s%s(SB)", dot, f.Name)
	if f.Attributes != 0 {
		p.Printf(", %s", f.Attributes.Asm())
	}
	p.Printf(", %s\n", textsize(f))

	for _, node := range f.Nodes {
		switch n := node.(type) {
		case *avo.Instruction:
			if len(n.Operands) > 0 {
				p.Printf("\t%s\t%s\n", n.Opcode, joinOperands(n.Operands))
			} else {
				p.Printf("\t%s\n", n.Opcode)
			}
		case avo.Label:
			p.Printf("%s:\n", n)
		default:
			panic("unexpected node type")
		}
	}
}

func (p *goasm) global(g *avo.Global) {
	p.NL()
	for _, d := range g.Data {
		a := operand.NewDataAddr(g.Symbol, d.Offset)
		p.Printf("DATA %s/%d, %s\n", a.Asm(), d.Value.Bytes(), d.Value.Asm())
	}
	p.Printf("GLOBL %s(SB), %s, $%d\n", g.Symbol, g.Attributes.Asm(), g.Size)
}

func textsize(f *avo.Function) string {
	// Reference: https://github.com/golang/go/blob/b115207baf6c2decc3820ada4574ef4e5ad940ec/src/cmd/internal/obj/util.go#L260-L265
	//
	//		case TYPE_TEXTSIZE:
	//			if a.Val.(int32) == objabi.ArgsSizeUnknown {
	//				str = fmt.Sprintf("$%d", a.Offset)
	//			} else {
	//				str = fmt.Sprintf("$%d-%d", a.Offset, a.Val.(int32))
	//			}
	//
	s := "$" + strconv.Itoa(f.FrameBytes())
	if argsize := f.ArgumentBytes(); argsize > 0 {
		return s + "-" + strconv.Itoa(argsize)
	}
	return s
}

func joinOperands(operands []operand.Op) string {
	asm := make([]string, len(operands))
	for i, op := range operands {
		asm[i] = op.Asm()
	}
	return strings.Join(asm, ", ")
}