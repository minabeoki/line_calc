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

func printAst(tree ast.Expr) {
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

func expr(op string, x *big.Float, y *big.Float) *big.Float {
	z := new(big.Float)
	switch op {
	case "+":
		z = z.Add(x, y)
	case "-":
		z = z.Sub(x, y)
	case "*":
		z = z.Mul(x, y)
	case "/":
		z = z.Quo(x, y)
	case "<<":
	case ">>":
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
			z := expr(op, x, y)
			stack = append(stack, z)
		case *ast.BasicLit:
			lit := node.(*ast.BasicLit)
			x := new(big.Float)
			fmt.Sscan(lit.Value, x)
			stack = append(stack, x)
		}
	}

	if len(stack) == 0 {
		return nil
	}
	return stack[0]
}

func answer(line string) (string, error) {
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
	defer rl.Close()

	rl.Config.SetListener(keyListener)

	fmt.Println()
	for {
		_, err := rl.Readline()
		if err != nil {
			break
		}
		fmt.Println()
	}
}

const (
	escDown  = "\x1bD"
	escUp    = "\x1bM"
	escEnter = "\x1bE"
	escKill  = "\x1b[K"
	escLeft  = "\x1b[%dD"
	escRight = "\x1b[%dC"
)

var prev = ""

func keyListener(line []rune, pos int, key rune) ([]rune, int, bool) {
	if key == '\n' || key == '\r' || key == 0x04 {
		return line, pos, true
	}

	ans, _ := answer(string(line))

	if ans != prev {
		fmt.Printf(escUp+escLeft+escKill, pos+2)
		fmt.Printf("[ %s ]\n", ans)
		prev = ans
	}

	return line, pos, true
}

/// line_calc.go ends here
