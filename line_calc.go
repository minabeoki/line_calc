/// line_calc.go ---

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"math/big"
	"strings"

	"github.com/chzyer/readline"
)

const precision = 72

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
		"k", "*1024",
		"K", "*1024",
		"m", "*1024*1024",
		"M", "*1024*1024",
		"g", "*1024*1024*1024",
		"G", "*1024*1024*1024",
		"~", "!",
		"pi", "3.14159265358979323846264338327950",
	)
	return replacer.Replace(line)
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
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer func() { _ = rl.Close() }()

	rl.Config.SetListener(keyListener)

	fmt.Println()
	for {
		_, err := rl.Readline()
		if err != nil {
			break
		}
		fmt.Println(escClear)
	}
}

const (
	escDown1 = "\x1bD"
	escUp1   = "\x1bM"
	escEnter = "\x1bE"
	escKill  = "\x1b[K"
	escClear = "\x1b[2K"
	escUp    = "\x1b[%dA"
	escLeft  = "\x1b[%dD"
	escRight = "\x1b[%dC"
)

var prev = ""

func keyListener(line []rune, pos int, key rune) ([]rune, int, bool) {
	if key == '\n' || key == '\r' || key == 0x04 {
		prev = ""
		return line, pos, true
	}

	ans, _ := answer(string(line))

	if ans != prev {
		fmt.Printf(escUp1+escLeft, pos+2)
		fmt.Printf("[ %s ]", ans)
		fmt.Println(escKill)
		prev = ans
	}

	return line, pos, true
}

/// line_calc.go ends here
