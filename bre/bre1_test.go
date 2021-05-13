package bre

import (
	"fmt"
	"testing"
)

func TestExe1(t *testing.T) {
	t.Parallel()

	pkgCode := "Singapore.Footwear.Test 100 Rules"
	sbu := "adidas"
	facts := []byte(`{"sku":"ADKFZ1764YEL2..","price":"100","flag":"1","trace":""}`)

	results, err := ExeBrePkg(pkgCode, facts, &sbu)
	if err != nil {
		t.Logf("Failedin Execution %s", err)
		t.Fail()
	}

	if results["price"] != "60" {
		t.Logf("Failedin Return Pri,Should be 60 but got %s", results["price"])
		t.Fail()
	}

	if results["flag"] != "41" {
		t.Logf("Failed in Return FlShould be 41 but got %s", results["flag"])
		t.Fail()
	}

}

func TestParallel(t *testing.T) {

	// pkgCode := "Singapore.Footwear.Test 100 Rules"
	// user := structs.User{Sbu: "adidas"}

	// err := LoadBrePkg(pkgCode, &user)
	// if err != nil {
	// 	t.Logf("FailedLoading BRE Package %s", err)
	// 	t.Fail()
	// }

	t.Run("group", func(t *testing.T) {

		for i := 1; i <= 100; i++ {
			t.Run(fmt.Sprintf("Test%d", i), TestExe1)
		}

	})
}
