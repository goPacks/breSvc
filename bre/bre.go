package bre

import (
	"breSvc/mongosvc"
	"breSvc/structs"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

type astNode struct {
	name       string
	expr       ast.Expr
	actionExpr []ast.Expr
}

var astNodes map[string]*astNode
var filters map[string]struct{}
var brePackage structs.BrePkg

func SetBrePkg(pBrePackage []byte, user *structs.User) (success bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Compile Error: %s", r)
		}
	}()

	// Zero Byte struct for MAP with Key and no Value
	var empty struct{}

	// Create Dimensions
	filters = make(map[string]struct{})

	// Unmarshall
	if err := json.Unmarshal(pBrePackage, &brePackage); err != nil {
		panic(err)
	}

	// Setp the dimensions
	for _, v := range brePackage.Filters {
		filters[v] = empty
	}

	// Create AST nodes
	compileErr := compile(&brePackage)
	if compileErr != nil {
		panic(compileErr)
	}

	// Save to Database
	_, saveErr := mongosvc.Upsert(brePackage, user)
	if saveErr != nil {
		panic(saveErr)
	}

	return true, nil
}

// With the facts provide, iterate through all the rules and corresponding actions in the ruleset.
func ExeBrePkg(pkgCode string, factBody []byte, user *structs.User) (results map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	// Setup facts collection to store existing table, modify and add new facts
	var facts map[string]string

	// Import facts received into facts collection
	if err := json.Unmarshal(factBody, &facts); err != nil {
		panic(err)
	}

	// Start trace
	facts["trace"] = ""

	if brePackage.PkgCode != pkgCode {

		brePkg, err := mongosvc.GetBrePkg(pkgCode, user)
		if err != nil {
			panic(err)
		}

		brePkgStr, err := json.Marshal(brePkg)
		if err != nil {
			panic(err)
		}

		_, err = SetBrePkg(brePkgStr, user)
		if err != nil {
			panic(err)
		}
	}

	// Traverse through all rules in the ruleset
	for _, v := range brePackage.RuleSet {
		err := exeAstNodes(v.RuleName, v.Actions, &facts, &filters)
		if err != nil {
			panic(err)
		}
	}

	return facts, nil
}

// Parse the BRE package into AST nodes
func compile(brePackage *structs.BrePkg) (err error) {

	errLevel := "Rule"

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s - %s", errLevel, r)
		}
	}()

	// Create dictionary to store the AST nodes
	astNodes = make(map[string]*astNode)

	errLevel = "Action"

	for _, rule := range brePackage.RuleSet {
		ruleExpr, ruleErr := parser.ParseExpr(rule.Rule)
		if ruleErr != nil {
			panic(fmt.Sprintf("%s - %s", rule.Rule, ruleErr))
		}

		// spew.Dump(ruleExpr)

		astNodes[rule.RuleName] = &astNode{name: rule.RuleName, expr: ruleExpr}

		for _, action := range rule.Actions {

			cmd := strings.ReplaceAll(action, "=", "==")

			actionExpr, actionErr := parser.ParseExpr(cmd)

			// For Debugging Purpose
			//	spew.Dump(actionExpr)

			if actionErr != nil {
				panic(fmt.Sprintf("%s - %s", action, actionErr))
			}

			astNodes[rule.RuleName].actionExpr = append(astNodes[rule.RuleName].actionExpr, actionExpr)

			// json_data2, err := json.Marshal(actionExpr)

			// if err != nil {
			// 	log.Fatal(err)
			// }

			// fmt.Println(string(json_data2))

			// var astExpr ast.Expr

			// err = json.Unmarshal(json_data2, &astExpr)

			// if err != nil {
			// 	fmt.Printf("%s", err)
			// }

		}

	}

	return nil
}

func exeAstNodes(ruleName string, actions []string, facts *map[string]string, filters *map[string]struct{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in exeAstNodes : %s", r)
		}
	}()

	astNode := astNodes[ruleName]
	rule := astNode.expr

	ruleOk, err := eval(rule, true, true, 0, facts, filters)
	if err != nil {
		panic(err)
	}

	if ruleOk == "true" {

		(*facts)["trace"] = (*facts)["trace"] + ruleName + ";"

		for _, action := range astNode.actionExpr {
			//	eval(action, false, true, 0, facts, filters)
			_, err := eval(action, false, true, 0, facts, filters)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func eval(exp ast.Expr, isRule bool, isLeft bool, cnt int, facts *map[string]string, filters *map[string]struct{}) (node string, err error) {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:

		node, err := evalBinaryExpr(exp, isRule, isLeft, cnt, facts, filters)
		if err != nil {
			return "", err
		}

		return node, nil

	case *ast.BasicLit:
		switch exp.Kind {
		case token.INT:
			return exp.Value, nil
		case token.STRING:
			return exp.Value, nil
		case token.FLOAT:
			return exp.Value, nil
		}
	case *ast.ParenExpr:
		return eval(exp.X, isRule, isLeft, cnt, facts, filters)
	case *ast.Ident:

		// Assignment
		if isRule {
			return exp.Name, nil

		} else {
			if strings.HasPrefix(exp.Name, "_") {
				return exp.Name[1:], nil
			} else {
				v, exist := (*facts)[exp.Name]
				if exist {
					return fmt.Sprintf("%v", v), nil
				} else {
					return exp.Name, nil
				}
			}

		}
	}
	return "", nil
}

func evalBinaryExpr(exp *ast.BinaryExpr, isRule bool, isLeft bool, cnt int, facts *map[string]string, filters *map[string]struct{}) (node string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in evalBinaryExprn : %s", r)
		}
	}()

	left, err := eval(exp.X, isRule, true, cnt+1, facts, filters)
	if err != nil {
		return "", err
	}

	right, err := eval(exp.Y, isRule, false, cnt+1, facts, filters)
	if err != nil {
		return "", err
	}

	switch exp.Op {
	case token.ADD:
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat + rightFloat
		return fmt.Sprintf("%.2f", ans), nil

	case token.SUB:
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat - rightFloat

		return fmt.Sprintf("%.2f", ans), nil

	case token.MUL:
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat * rightFloat

		return fmt.Sprintf("%.2f", ans), nil

	case token.QUO:
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat / rightFloat

		return fmt.Sprintf("%.2f", ans), nil

	case token.LAND:
		if left == "true" && right == "true" {
			return "true", nil
		} else {
			return "false", nil
		}
	case token.LOR:
		if left == "true" || right == "true" {
			return "true", nil
		} else {
			return "false", nil
		}

	case token.NEQ:
		if strings.HasPrefix(right, "xls") {
			v, exist := (*facts)[left]
			if exist {
				key := fmt.Sprintf("%s-%v", right, v)
				_, x := (*filters)[key]
				if x {
					return "true", nil
				} else {
					return "false", nil
				}

			} else {
				return "false", nil
			}
		} else {

			isEql := (*facts)[left] != right

			if isEql {
				return "true", nil
			} else {
				return "false", nil
			}

		}

	case token.EQL:
		// Rule or Action
		if isRule {
			// Check Dimension
			if strings.HasPrefix(right, "xls") {
				v, exist := (*facts)[left]
				if exist {
					key := fmt.Sprintf("%s-%v", right, v)
					_, x := (*filters)[key]
					if x {
						return "true", nil
					} else {
						return "false", nil
					}

				} else {
					return "false", nil
				}
			} else {

				isEql := (*facts)[left] == right

				if isEql {
					return "true", nil
				} else {
					return "false", nil
				}

			}

		} else {
			// Assignment
			(*facts)[string(left)] = right
		}

	}

	return "", nil
}

func ToFloat64(left, right string) (float64, float64, error) {
	floatLeft, err := strconv.ParseFloat(left, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to convert %v to float, %s", left, err)
	}

	floatRight, err := strconv.ParseFloat(right, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to convert %v to float, %s", right, err)
	}

	return floatLeft, floatRight, nil
}
