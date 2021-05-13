// This package provides the BRE functionality.
// Rules are sotred in BRE containers, I call brePkg
//

package bre

import (
	"breSvc/mongosvc"
	"breSvc/structs"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"
)

// Collection cache of all brePkgs
var brePkgs []*structs.BrePkg

// Function saves a brePkg to data storage. Before saving it compiles the package to check for syntax errors and cachses the compiled AST nodes
func SaveBrePkg(pBrePkgReq []byte, sbu *string) (success bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Compile Error: %s", r)
		}
	}()

	var brePkgReq structs.BrePkgReq // For data storage
	var brePkg structs.BrePkg       // AST Nodes

	// Unmarshall for data storage
	if err := json.Unmarshal(pBrePkgReq, &brePkgReq); err != nil {
		panic(err)
	}

	// Compile and check for syntax error
	compileErr := compile(&brePkgReq, &brePkg)
	if compileErr != nil {
		panic(compileErr)
	}

	// Cache the package for later use
	brePkgs = append(brePkgs, &brePkg)

	// Make key
	pkgId := fmt.Sprintf("%s.%s.%s", brePkgReq.Site, brePkgReq.Cat, brePkgReq.PkgCode)

	// Delete all recorded dimensions for the key
	delErr := mongosvc.DelDim(pkgId, sbu)
	if delErr != nil {
		panic(delErr)
	}

	// Insert Individual Dimensions
	for _, v := range brePkgReq.Dimensions {
		_, saveErr := mongosvc.InsDim(pkgId, v, sbu)
		if saveErr != nil {
			panic(saveErr)
		}
	}

	brePkgReq.Dimensions = nil

	// Save to databsae
	_, saveErr := mongosvc.UpsertBre(&brePkgReq, sbu)
	if saveErr != nil {
		panic(saveErr)
	}

	return true, nil
}

// Function extracts brePks from data stroage, compiles cachses the compiled AST nodes
func LoadBrePkg(pkgCode string, sbu *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)

		}
	}()

	brePkgReq, err := mongosvc.GetBrePkg(pkgCode, sbu)
	if err != nil {
		panic(err)
	}

	var brePkg structs.BrePkg

	// Compile and check for syntax errors
	compileErr := compile(&brePkgReq, &brePkg)
	if compileErr != nil {
		panic(compileErr)
	}

	// Cache the package for later use
	brePkgs = append(brePkgs, &brePkg)

	return nil
}

// Function to check if dimension exists in storage
func chkDimExist(pkgCode *string, dim *string, sbu *string) (exist bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	// Extract from storage
	dimData, err := mongosvc.GetDim(pkgCode, dim, sbu)
	if err != nil {
		panic(err)
	}

	// Reutunr true if exist
	if dimData.Data != "" {
		return true, nil
	} else {
		return false, nil
	}
}

// With the facts provided, iterate through all the rules and corresponding actions in the ruleset.
func ExeBrePkg(pkgCode string, factBody []byte, sbu *string) (results map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	// Setup facts collection to store existing table, modify and add new facts
	var facts map[string]string
	var pkgBre *structs.BrePkg

	// Import facts received into facts collection
	if err := json.Unmarshal(factBody, &facts); err != nil {
		panic(err)
	}

	pkgInMemory := false

	// Start trace
	facts["trace"] = ""

	// Check if package exist in memory
	for _, pkg := range brePkgs {

		if pkg.PkgCode == pkgCode {
			pkgBre = pkg
			pkgInMemory = true
			break
		}
	}

	if pkgInMemory == false {

		if err := LoadBrePkg(pkgCode, sbu); err != nil {
			panic(err)
		}

		for _, pkg := range brePkgs {

			if pkg.PkgCode == pkgCode {
				pkgBre = pkg
				pkgInMemory = true
				break

			}

		}

	}

	if pkgInMemory == false {
		return facts, fmt.Errorf("Bre Package %s not stored", pkgCode)
	}

	// Traverse and evluate all the nodes
	for _, v := range pkgBre.AstNodes {
		err := exeAstNodes(pkgBre.PkgCode, v.Name, v, &facts, sbu)
		if err != nil {
			panic(err)
		}
	}

	return facts, nil

}

// Parse the BRE package into AST nodes
func compile(brePkgReq *structs.BrePkgReq, brePkg *structs.BrePkg) (err error) {

	errLevel := "Rule"

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s - %s", errLevel, r)
		}
	}()

	// Create ID
	brePkg.PkgCode = brePkgReq.Site + "." + brePkgReq.Cat + "." + brePkgReq.PkgCode
	brePkg.Cat = brePkgReq.Cat
	brePkg.Site = brePkgReq.Site

	brePkg.AstNodes = make([]*structs.AstNode, 0)

	errLevel = "Action"

	for _, rule := range brePkgReq.RuleSet {
		ruleExpr, ruleErr := parser.ParseExpr(rule.Rule)
		if ruleErr != nil {
			panic(fmt.Sprintf("%s - %s", rule.Rule, ruleErr))
		}

		// For Debugging purpose if needed
		// spew.Dump(ruleExpr)

		brePkg.AstNodes = append(brePkg.AstNodes, &structs.AstNode{Name: rule.RuleName, Expr: ruleExpr})

		for _, action := range rule.Actions {

			cmd := strings.ReplaceAll(action, "=", "==")

			actionExpr, actionErr := parser.ParseExpr(cmd)

			// For Debugging purpose if needed
			//	spew.Dump(actionExpr)

			if actionErr != nil {
				panic(fmt.Sprintf("%s - %s", action, actionErr))
			}

			brePkg.AstNodes[len(brePkg.AstNodes)-1].ActionExpr = append(brePkg.AstNodes[len(brePkg.AstNodes)-1].ActionExpr, actionExpr)

		}

	}

	return nil
}

// Traverse through all nodes and execute
func exeAstNodes(pkgId string, ruleName string, astNode *structs.AstNode, facts *map[string]string, sbu *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in exeAstNodes : %s", r)
		}
	}()

	rule := astNode.Expr

	ruleOk, err := eval(&pkgId, rule, true, facts, sbu)
	if err != nil {
		panic(err)
	}

	if ruleOk == "true" {

		(*facts)["trace"] = (*facts)["trace"] + ruleName + ";"

		for _, action := range astNode.ActionExpr {
			//	eval(action, false, true, 0, facts, filters)
			_, err := eval(&pkgId, action, false, facts, sbu)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func eval(pkgId *string, exp ast.Expr, isRule bool, facts *map[string]string, sbu *string) (node string, err error) {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:

		node, err := evalBinaryExpr(pkgId, exp, isRule, facts, sbu)
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
		return eval(pkgId, exp.X, isRule, facts, sbu)
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

// Evaluate Binary Expression
func evalBinaryExpr(pkgId *string, exp *ast.BinaryExpr, isRule bool, facts *map[string]string, sbu *string) (node string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in evalBinaryExprn : %s", r)
		}
	}()

	left, err := eval(pkgId, exp.X, isRule, facts, sbu)
	if err != nil {
		return "", err
	}

	right, err := eval(pkgId, exp.Y, isRule, facts, sbu)
	if err != nil {
		return "", err
	}

	switch exp.Op {
	case token.ADD:
		leftFloat, rightFloat, err := toFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat + rightFloat

		return str(ans), nil

	case token.SUB:
		leftFloat, rightFloat, err := toFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat - rightFloat

		return str(ans), nil

	case token.MUL:
		leftFloat, rightFloat, err := toFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat * rightFloat

		return str(ans), nil

	case token.QUO:
		leftFloat, rightFloat, err := toFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat / rightFloat

		return str(ans), nil

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

				exist, err := chkDimExist(pkgId, &key, sbu)
				if err != nil {
					panic(err)
				}

				if exist {
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

					exist, err := chkDimExist(pkgId, &key, sbu)
					if err != nil {
						panic(err)
					}

					if exist {
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

func toFloat64(left, right string) (float64, float64, error) {
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

func str(nbr float64) string {

	if math.Mod(nbr, 1.0) == 0 {
		return fmt.Sprintf("%v", nbr)
	} else {
		return fmt.Sprintf("%.2f", nbr)
	}
}
