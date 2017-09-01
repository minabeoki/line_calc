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
	precision = 128
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
		"**", "^",
		"pi", "3.14159265358979323846264338327950",
	)
	s := replacer.Replace(line)

	const patNum = `(0x[0-9a-fA-F]+|[0-9]+)`
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

func answer(line string) (s []string, err error) {
	line = preconv(line)
	tree, err := parser.ParseExpr(line)
	if err != nil {
		return s, err
	}
	//printAst(tree)
	rpn := traverse(tree)
	ans := calc(rpn)
	if ans == nil {
		return s, nil
	}

	if ans.IsInt() {
		v, _ := ans.Int(nil)
		s = append(s, separater(v.Text(10), ",", 3))

		minus := ""
		z := new(big.Int)
		z.SetUint64(0)
		if v.Cmp(z) < 0 {
			if v.BitLen() <= 32 {
				z.SetBit(z, 32, 1)
				v = z.Add(z, v)
			} else if v.BitLen() <= 64 {
				z.SetBit(z, 64, 1)
				v = z.Add(z, v)
			} else {
				minus = "-"
				v.Abs(v)
			}
		}

		s = append(s, minus+"0x"+separater(v.Text(16), "_", 4))
		s = append(s, minus+"0b"+separater(v.Text(2), "_", 8))
	} else {
		//s = append(s, ans.Text('f', 16))
		s = append(s, fmt.Sprint(ans))
	}

	return s, nil
}

func separater(num string, sep string, n int) string {
	r := ""
	for i := 0; i < len(num); i++ {
		c := string(num[len(num)-i-1])
		if i > 0 && (i%n) == 0 && c != "-" {
			c += sep
		}
		r = c + r
	}
	return r
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
		printAns(ans)
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

func printAns(ans []string) int {
	if len(ans) == 0 {
		return 0
	}

	n := 1
	out := aprompt + ans[0]
	if len(ans) >= 2 {
		out += "  " + ans[1]
	}
	out += escKill + "\n"

	if len(ans) >= 3 {
		for i := 0; i < len(aprompt); i++ {
			out += " "
		}
		out += ans[2]
	}

	n++
	fmt.Println(out + escKill)

	return n
}

func keyListener(line []rune, pos int, key rune) ([]rune, int, bool) {
	switch key {
	case '\n', '\r', 0x04, 0:
		// do nothing
	default:
		ans, _ := answer(string(line))

		fmt.Print(escEnter)
		n := printAns(ans)
		out := fmt.Sprintf(escUp, n+1)
		out += fmt.Sprintf(escRight, len(prompt)+pos)
		fmt.Print(out)
	}

	return nil, 0, false
}

/// line_calc.go ends here
