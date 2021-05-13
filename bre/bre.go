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

var brePkgs []*structs.BrePkg

func SetBrePkg(pBrePkgReq []byte, sbu *string) (success bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Compile Error: %s", r)
		}
	}()

	// Zero Byte struct for MAP with Key and no Value
	//var empty struct{}
	var brePkgReq structs.BrePkgReq
	//var brePkg structs.BrePkg

	// Unmarshall
	//	if err := json.Unmarshal(pBrePkgReq, &brePkgReq); err != nil {
	//		panic(err)
	//	}

	// Create AST nodes
	// compileErr := compile(&brePkgReq, &brePkg)
	// if compileErr != nil {
	// 	panic(compileErr)
	// }

	// Save Dimensions to DB

	//	brePkg.Filters = make(map[string]struct{})
	_, saveErr := mongosvc.UpsertBre(&brePkgReq, sbu)
	if saveErr != nil {
		panic(saveErr)
	}

	//brePkgs = append(brePkgs, &brePkg)

	pkgId := fmt.Sprintf("%s.%s.%s", brePkgReq.Site, brePkgReq.Cat, brePkgReq.PkgCode)

	delErr := mongosvc.DelDim(pkgId, sbu)
	if delErr != nil {
		panic(delErr)
	}

	for _, v := range brePkgReq.Filters {
		//	_, saveErr := mongosvc.InsDim(&structs.Dim{Data: v}, brePkgReq.Site, brePkgReq.Cat, brePkgReq.PkgCode, user)

		_, saveErr := mongosvc.InsDim(pkgId, v, sbu)
		if saveErr != nil {
			panic(saveErr)
		}
	}

	// Setp the dimensions
	// for _, v := range brePkgReq.Filters {
	// 	brePkg.Filters[v] = empty
	// }

	// brePkgs = append(brePkgs, &brePkg)

	// Save to Database
	// _, saveErr := mongosvc.UpsertBre(&brePkgReq, user)
	// if saveErr != nil {
	// 	panic(saveErr)
	// }

	return true, nil
}

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

	compileErr := compile(&brePkgReq, &brePkg)
	if compileErr != nil {
		panic(compileErr)
	}

	brePkgs = append(brePkgs, &brePkg)

	// brePkgStr, err := json.Marshal(brePkg)
	// if err != nil {
	// 	panic(err)
	// }

	// _, err = SetBrePkg(brePkgStr, sbu)
	// if err != nil {
	// 	panic(err)
	// }

	return nil
}

func chkDimExist(pkgCode *string, dim *string, sbu *string) (exist bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	dimData, err := mongosvc.GetDim(pkgCode, dim, sbu)
	if err != nil {
		panic(err)
	}

	if dimData.Data != "" {
		return true, nil
	} else {
		return false, nil
	}

}

// With the facts provide, iterate through all the rules and corresponding actions in the ruleset.
func ExeBrePkg(pkgCode string, factBody []byte, sbu *string) (results map[string]string, err error) {
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

	pkgInMemory := false
	var pkgBre *structs.BrePkg

	// Start trace
	facts["trace"] = ""

	// Check if package exist in memeory
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

				// 	for _, v := range pkg.AstNodes {

				// 		err := exeAstNodes(v.Name, v, &facts, &pkg.Filters)
				// 		if err != nil {
				// 			panic(err)
				// 		}
				// 	}

				// 	return facts, nil
				// }
			}

		}

	}

	if pkgInMemory == false {
		return facts, fmt.Errorf("Bre Package %s not stored", pkgCode)
	}
	// -----------------------------------

	//---------------------
	for _, v := range pkgBre.AstNodes {
		//	pkgId := fmt.Sprintf("%s.%s.%s", pkgBre.Site, pkgBre.Cat, pkgBre.PkgCode)

		err := exeAstNodes(pkgBre.PkgCode, v.Name, v, &facts, sbu)
		if err != nil {
			panic(err)
		}
	}

	return facts, nil

}

// if brePackage.PkgCode != pkgCode {

// 	//	log.Println("Loading pack")

// 	if err := LoadBrePkg(pkgCode, user); err != nil {
// 		panic(err)
// 	}

// }

// //Traverse through all rules in the ruleset
// for _, v := range brePackage.RuleSet {
// 	err := exeAstNodes(v.RuleName, v.Actions, &facts, &filters)
// 	if err != nil {
// 		panic(err)
// 	}
// }

//}
// Parse the BRE package into AST nodes
func compile(brePkgReq *structs.BrePkgReq, brePkg *structs.BrePkg) (err error) {

	errLevel := "Rule"

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s - %s", errLevel, r)
		}
	}()

	// Create dictionary to store the AST nodes

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

		// spew.Dump(ruleExpr)

		//	brePkg.AstNodes[rule.RuleName] = &structs.AstNode{Name: rule.RuleName, Expr: ruleExpr}
		brePkg.AstNodes = append(brePkg.AstNodes, &structs.AstNode{Name: rule.RuleName, Expr: ruleExpr})

		for _, action := range rule.Actions {

			cmd := strings.ReplaceAll(action, "=", "==")

			actionExpr, actionErr := parser.ParseExpr(cmd)

			// For Debugging Purpose
			//	spew.Dump(actionExpr)

			if actionErr != nil {
				panic(fmt.Sprintf("%s - %s", action, actionErr))
			}

			//astNodes[rule.RuleName].actionExpr = append(astNodes[rule.RuleName].actionExpr, actionExpr)

			//brePkg.AstNodes[rule.RuleName].ActionExpr = append(brePkg.AstNodes[rule.RuleName].ActionExpr, actionExpr)
			brePkg.AstNodes[len(brePkg.AstNodes)-1].ActionExpr = append(brePkg.AstNodes[len(brePkg.AstNodes)-1].ActionExpr, actionExpr)

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

func exeAstNodes(pkgId string, ruleName string, astNode *structs.AstNode, facts *map[string]string, sbu *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in exeAstNodes : %s", r)
		}
	}()

	//astNode := astNodes[ruleName]
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
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat + rightFloat

		return str(ans), nil

	case token.SUB:
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat - rightFloat

		return str(ans), nil

	case token.MUL:
		leftFloat, rightFloat, err := ToFloat64(left, right)
		if err != nil {
			panic(err)
		}

		ans := leftFloat * rightFloat

		return str(ans), nil

	case token.QUO:
		leftFloat, rightFloat, err := ToFloat64(left, right)
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
					// _, x := (*filters)[key]
					// if x {
					// 	return "true", nil
					// } else {
					// 	return "false", nil
					// }

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

func str(nbr float64) string {

	if math.Mod(nbr, 1.0) == 0 {
		return fmt.Sprintf("%v", nbr)
	} else {
		return fmt.Sprintf("%.2f", nbr)
	}
}
