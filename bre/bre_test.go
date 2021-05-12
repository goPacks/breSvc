package bre

import (
	"breSvc/structs"
	"fmt"
	"testing"
)

func TestExe(t *testing.T) {
	t.Parallel()

	pkgCode := "Singapore.Footwear.Test 100 Rules"
	user := structs.User{Sbu: "adidas"}
	facts := []byte(`{"sku":"ADKFZ1153WHT1..","price":"100","flag":"1","trace":""}`)

	results, err := ExeBrePkg(pkgCode, facts, &user)
	if err != nil {
		t.Logf("Failed in Execution %s", err)
		t.Fail()
	}

	if results["price"] != "60" {
		t.Logf("Failed in Return Pri, Should be 60 but got %s", results["price"])
		t.Fail()
	}

	if results["flag"] != "41" {
		t.Logf("Failed in Return Fl, Should be 41 but got %s", results["flag"])
		t.Fail()
	}

	//log.Println(results)
}

func TestInParallel(t *testing.T) {
	// This Run will -not return until e paallel tests finish.

	t.Run("group", func(t *testing.T) {

		for i := 1; i <= 10; i++ {
			t.Run(fmt.Sprintf("Test%d", i), TestExe)
		}
		//t.Run("Test1", TestExe
		// t.Run("Test2", TestE)
		// t.Run("Test3", TestExe
	})

}
