/// line_calc.go ---

package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"math"
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

var tblIdent = map[string]*big.Float{}

var units = map[string]int64{
	"K": 1024,
	"M": 1024 * 1024,
	"G": 1024 * 1024 * 1024,
	"T": 1024 * 1024 * 1024 * 1024,
	"k": 1000,
	"m": 1000 * 1000,
	"g": 1000 * 1000 * 1000,
	"t": 1000 * 1000 * 1000 * 1000,
}

func preconv(line string) string {
	replacer := strings.NewReplacer(
		"~", "!",
		"**", "^",
		"pi", "3.14159265358979323846264338327950",
	)
	s := replacer.Replace(line)

	rs := `([`
	for k := range units {
		rs += k
	}
	rs += `])`

	re := regexp.MustCompile(rs)
	s = re.ReplaceAllString(s, ".($1)")

	return s
}

func operation1(op string, x *big.Float) (z *big.Float, err error) {
	switch op {
	case "+":
		z = x
	case "-":
		z = x.Neg(x)
	case "!":
		p, _ := x.Int(nil)
		z = x.SetInt(p.Not(p))
	default:
		err = errors.New("invalid unary")
	}
	return z, err
}

func operation2(op string, x, y *big.Float) (z *big.Float, err error) {
	var r *big.Int
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
		r = p.Mod(p, q)
	case "^":
		r = p.Exp(p, q, nil)
	case "<<":
		r = p.Lsh(p, uint(q.Int64()))
	case ">>":
		r = p.Rsh(p, uint(q.Int64()))
	case "&":
		r = p.And(p, q)
	case "|":
		r = p.Or(p, q)
	default:
		err = errors.New("invalid op")
	}

	if r != nil {
		z = x.SetInt(r)
	}

	return z, err
}

func evalExpr(expr ast.Expr) (*big.Float, error) {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return evalExpr(e.X)
	case *ast.BinaryExpr:
		return evalBinaryExpr(e)
	case *ast.UnaryExpr:
		return evalUnaryExpr(e)
	case *ast.BasicLit:
		x, _, err := big.ParseFloat(e.Value, 10, precision, big.ToNearestEven)
		return x, err
	case *ast.Ident:
		return evalIdent(e)
	case *ast.CallExpr:
		return evalCallExpr(e)
	case *ast.TypeAssertExpr:
		return evalUnit(e.X, e.Type)
	}

	return nil, errors.New("invalid expr")
}

func evalBinaryExpr(expr *ast.BinaryExpr) (*big.Float, error) {
	x, err := evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	y, err := evalExpr(expr.Y)
	if err != nil {
		return nil, err
	}

	return operation2(expr.Op.String(), x, y)
}

func evalUnaryExpr(expr *ast.UnaryExpr) (*big.Float, error) {
	x, err := evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	return operation1(expr.Op.String(), x)
}

func evalIdent(expr *ast.Ident) (*big.Float, error) {
	v, ok := tblIdent[expr.Name]
	if !ok {
		return nil, errors.New("unknown ident")
	}
	return v, nil
}

func evalCallExpr(expr *ast.CallExpr) (*big.Float, error) {
	if len(expr.Args) == 0 {
		return nil, errors.New("no args")
	}

	switch e := expr.Fun.(type) {
	case *ast.Ident:
		var args []float64
		for _, e := range expr.Args {
			v, err := evalExpr(e)
			if err != nil {
				return nil, err
			}
			a, _ := v.Float64()
			args = append(args, a)
		}

		switch e.Name {
		case "sqrt":
			x := big.NewFloat(math.Sqrt(args[0]))
			x.SetPrec(precision)
			return x, nil
		}

		return nil, errors.New("unknown call " + e.Name)

	case *ast.BasicLit:
		return evalUnit(e, expr.Args[0])
	}

	return nil, errors.New("invalid call")
}

func evalUnit(expr, unit ast.Expr) (*big.Float, error) {
	u, ok := unit.(*ast.Ident)
	if !ok {
		return nil, errors.New("invalid unit")
	}

	x, err := evalExpr(expr)
	if err != nil {
		return nil, err
	}

	v, ok := units[u.Name]
	if !ok {
		return x, errors.New("unknown unit")
	}

	z := new(big.Float).SetPrec(precision).SetInt64(v)
	z = x.Mul(x, z)

	return z, nil
}

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

func answer(line string) (s []string, err error) {
	line = preconv(line)
	tree, err := parser.ParseExpr(line)
	if err != nil {
		return s, err
	}
	//printAst(tree)
	ans, err := evalExpr(tree)
	if err != nil {
		return s, err
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
