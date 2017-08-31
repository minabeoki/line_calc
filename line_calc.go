/// line_calc.go ---

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"math/big"
	"regexp"
	"strings"

	"github.com/chzyer/readline"
)

const (
	prompt    = "> "
	aprompt   = "=> "
	precision = 72
	patNum    = `(0x[0-9a-fA-F]+|[0-9]+)`
)

func printAst(tree ast.Expr) {
	fmt.Println()
	depth := 0
	ast.Inspect(tree, func(n ast.Node) bool {
		indent := strings.Repeat("  ", depth)
		if n != nil {
			fmt.Printf("%s%[2]T %[2]v\n", indent, n)
			depth++
		} else {
			depth--
		}
		return true
	})
	fmt.Println()
}

func traverse(tree ast.Expr) (rpn []ast.Node) {
	ast.Inspect(tree, func(node ast.Node) bool {
		if node != nil {
			rpn = append(rpn, node)
		}
		return true
	})
	return rpn
}

func operation1(op string, x *big.Float) (z *big.Float) {
	switch op {
	case "+":
		z = x
	case "-":
		z = x.Neg(x)
	case "!":
		// what is correct??
		p, _ := x.Int(nil)
		r := p.Not(p)
		z = x.SetInt(r)
	default:
		z = x
	}
	return z
}

func operation2(op string, x, y *big.Float) (z *big.Float) {
	p, _ := x.Int(nil)
	q, _ := y.Int(nil)

	switch op {
	case "+":
		z = x.Add(x, y)
	case "-":
		z = x.Sub(x, y)
	case "*":
		z = x.Mul(x, y)
	case "/":
		z = x.Quo(x, y)
	case "%":
		r := p.Mod(p, q)
		z = x.SetInt(r)
	case "^":
		r := p.Exp(p, q, nil)
		z = x.SetInt(r)
	case "<<":
		r := p.Lsh(p, uint(q.Int64()))
		z = x.SetInt(r)
	case ">>":
		r := p.Rsh(p, uint(q.Int64()))
		z = x.SetInt(r)
	case "&":
		r := p.And(p, q)
		z = x.SetInt(r)
	case "|":
		r := p.Or(p, q)
		z = x.SetInt(r)
	}

	return z
}

func calc(rpn []ast.Node) *big.Float {
	var stack []*big.Float

	for i := len(rpn) - 1; i >= 0; i-- {
		node := rpn[i]
		switch node.(type) {
		case *ast.BinaryExpr:
			if len(stack) < 2 {
				return nil
			}
			x := stack[len(stack)-1]
			y := stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			op := node.(*ast.BinaryExpr).Op.String()
			z := operation2(op, x, y)
			stack = append(stack, z)
		case *ast.UnaryExpr:
			x := stack[len(stack)-1]
			op := node.(*ast.UnaryExpr).Op.String()
			z := operation1(op, x)
			stack[len(stack)-1] = z
		case *ast.BasicLit:
			lit := node.(*ast.BasicLit)
			x := new(big.Float).SetPrec(precision)
			fmt.Sscan(lit.Value, x)
			stack = append(stack, x)
		}
	}

	if len(stack) == 0 {
		return nil
	}
	return stack[0]
}

func preconv(line string) string {
	replacer := strings.NewReplacer(
		"~", "!",
		"pi", "3.14159265358979323846264338327950",
	)
	s := replacer.Replace(line)

	units := [][]string{
		{"K", "1024"},
		{"M", "1024*1024"},
		{"G", "1024*1024*1024"},
		{"T", "1024*1024*1024*1024"},
		{"k", "1000"},
		{"m", "1000*1000"},
		{"g", "1000*1000*1000"},
		{"t", "1000*1000*1000*1000"},
	}
	for _, u := range units {
		r := regexp.MustCompile(patNum + u[0])
		s = r.ReplaceAllString(s, "($1*"+u[1]+")")
	}

	return s
}

func answer(line string) (string, error) {
	line = preconv(line)
	tree, err := parser.ParseExpr(line)
	if err != nil {
		return "", err
	}
	//printAst(tree)
	rpn := traverse(tree)
	ans := calc(rpn)
	if ans == nil {
		return "", nil
	}

	s := ""
	if ans.IsInt() {
		v, _ := ans.Int(nil)
		s += v.Text(10)
		s += " 0x" + v.Text(16)
		s += " 0b" + v.Text(2)
	} else {
		s = fmt.Sprint(ans)
	}

	return s, nil
}

func main() {
	rl, err := readline.New(escBold + prompt + escNormal)
	if err != nil {
		panic(err)
	}
	defer func() { _ = rl.Close() }()

	rl.Config.SetListener(keyListener)

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		ans, _ := answer(line)
		fmt.Println(aprompt + ans)
	}
}

const (
	escDown1  = "\x1bD"
	escUp1    = "\x1bM"
	escEnter  = "\x1bE"
	escKill   = "\x1b[K"
	escClear  = "\x1b[2K"
	escUp     = "\x1b[%dA"
	escLeft   = "\x1b[%dD"
	escRight  = "\x1b[%dC"
	escNormal = "\x1b[0m"
	escBold   = "\x1b[1m"
)

func keyListener(line []rune, pos int, key rune) ([]rune, int, bool) {
	switch key {
	case '\n', '\r', 0x04, 0:
		// do nothing
	default:
		ans, _ := answer(string(line))
		out := escEnter + escKill
		out += aprompt + ans
		out += escUp1 + escEnter + escUp1
		out += fmt.Sprintf(escRight, len(prompt)+pos)
		fmt.Print(out)
	}

	return nil, 0, false
}

/// line_calc.go ends here
