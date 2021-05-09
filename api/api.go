package api

import (
	"breSvc/bre"
	"breSvc/jwt"
	"breSvc/mongosvc"
	"breSvc/structs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var (
	apiVer        = "/api/v2/"
	Port          string
	JwtExpiryTime time.Duration = 0
)

// Http Handlers to handle incoming API Calls
func HandleReq() {

	// Instantiate Router
	muxRouter := mux.NewRouter()

	// Security
	muxRouter.HandleFunc(apiVer+"logIn", logIn).Methods("POST")
	muxRouter.HandleFunc(apiVer+"regUser", regUser).Methods("POST")

	// Set Routes
	muxRouter.Handle(apiVer+"brePkg", chkJwt(setBrePkg)).Methods("PUT")
	muxRouter.Handle(apiVer+"brePkg/{pkgCode}", chkJwt(exeBrePkg)).Methods("POST")
	muxRouter.Handle(apiVer+"brePkg", chkJwt(getBrePkgs)).Methods("GET")
	muxRouter.Handle(apiVer+"brePkg/{pkgCode}", chkJwt(getBrePkg)).Methods("GET")
	muxRouter.Handle(apiVer+"brePkg/{pkgCode}", chkJwt(delBrePkg)).Methods("DELETE")
	muxRouter.Handle(apiVer+"brePkg", chkJwt(delBrePkgs)).Methods("DELETE")

	fmt.Println("Waiting on Port :" + Port)

	// err := http.ListenAndServeTLS(":"+Port, "cert/certificate.crt", "cert/privateKey.key", muxRouter)
	err := http.ListenAndServe(":"+Port, muxRouter)
	if err != nil {
		log.Fatal(err)

	}
}

// Intitializes the BRE package with rules and corresponding actions
func setBrePkg(w http.ResponseWriter, r *http.Request, user *structs.User) {

	// Read Message Body
	reqBody, err := ioutil.ReadAll(r.Body)

	// Error in Body so return error response
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("Please supply BRE package information in JSON format"))
		return
	}

	// Send Body to BRE
	success, err := bre.SetBrePkg(reqBody, user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(err.Error()))
		return
	}

	if success {

		// Return Respose
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("BRE Package Accepted"))
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("Unable to compile"))
	}
}

// Sends current facts to the rules engines for processing and returns the results
func exeBrePkg(w http.ResponseWriter, r *http.Request, user *structs.User) {

	params := mux.Vars(r)

	// Save to Database

	// Read Message Body
	reqBody, err := ioutil.ReadAll(r.Body)

	// Error in Body so return error response
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("422 -Please supply BRE facts information in JSON format"))
		return
	}

	// Send Body to BRE to Excute
	facts, err := bre.ExeBrePkg(params["pkgCode"], reqBody, user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(err.Error()))
		return
	} else {

		factsJson, err := json.Marshal(facts)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(factsJson))

	}
}

// Sends current facts to the rules engines for processing and returns the results
func getBrePkgs(w http.ResponseWriter, r *http.Request, user *structs.User) {

	// Save to Database
	brePkgs, err := mongosvc.GetAll(user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Unable to retreive : %s", err)))
		return

	}

	jsonBrePkgs, err := json.Marshal(brePkgs)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Unable to marshal Json Data : %s", err)))
		return

	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(jsonBrePkgs))
}

func getBrePkg(w http.ResponseWriter, r *http.Request, user *structs.User) {

	params := mux.Vars(r)

	brePkg, err := mongosvc.GetBrePkg(params["pkgCode"], user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Unable to retreive : %s", err)))
		return

	}

	jsonBrePkg, err := json.Marshal(brePkg)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Unable to marshal Json Data : %s", err)))
		return

	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(jsonBrePkg))
}

func delBrePkg(w http.ResponseWriter, r *http.Request, user *structs.User) {

	params := mux.Vars(r)

	// Save to Database
	err := mongosvc.Del(params["pkgCode"], user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Unable to Delete : %s", err)))
		return

	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Package Deleted"))
}

func delBrePkgs(w http.ResponseWriter, r *http.Request, user *structs.User) {

	// Save to Database
	err := mongosvc.DelAll(user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Unable to Delete : %s", err)))
		return

	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("All Packages Deleted"))
}

// Valildate JWT supplied
func chkJwt(endpoint func(http.ResponseWriter, *http.Request, *structs.User)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var token string

		// Get token from the Authorization header
		// Format: Authorization: Bearer
		tokens, ok := r.Header["Authorization"]
		if ok && len(tokens) >= 1 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// If the token is empty...
		if token == "" {
			// If we get here, the required token is missing
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		pLoad, err := jwt.ValidateJwt(token, jwt.SecretKey)

		if err != nil {
			errStr := fmt.Sprintf("{\"status\": -1, \"msg\": \"%s\", \"data\": \"%s\"}", "JWT Validation Failed", err)
			w.Write(([]byte(errStr)))

		} else {

			userData := structs.User{UserId: pLoad.UserId, Sbu: pLoad.Sbu}

			endpoint(w, r, &userData)
		}

	})
}

// Login Authentication
func logIn(w http.ResponseWriter, r *http.Request) {

	// Read Body of message
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("422 -Please supply " + "course information " + "in JSON format"))
		return
	}

	// Parse into User struct
	var user structs.User
	json.Unmarshal(reqBody, &user)
	if user.UserId == "" || user.PswdHash == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("422 -Please supply Userid & Password in JSON format"))
		return
	}

	// Check if UserId exists
	userData, err := mongosvc.GetUser(user.UserId)
	if userData.UserId == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("Invalid User Credentials"))
		return
	}

	if userData.PswdHash != user.PswdHash {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("Invalid User Credentials"))
		return
	}

	jwt := jwt.MakeJwt(JwtExpiryTime, jwt.PayLoad{UserId: user.UserId, Sbu: userData.Sbu})
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf("{\"jwt\":\"%s\"}", jwt)))
}

// Login Authentication
func regUser(w http.ResponseWriter, r *http.Request) {

	// Read Body of message
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("422 -Please supply Userid & Password in JSON format"))
		return
	}

	// Parse into User struct
	var user structs.User

	json.Unmarshal(reqBody, &user)
	if user.UserId == "" || user.PswdHash == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("422 -Please supply Userid & Password in JSON format"))
		return
	}

	if user.Sbu == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("422 -Please supply SBU in JSON format"))
		return
	}

	// Check if UserId exists
	userData, err := mongosvc.GetUser(user.UserId)
	if userData.UserId != "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("UserId Exsits"))
		return
	}

	_, err = mongosvc.RegUser(user.UserId, user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("UserId Registration Error : %s", err)))
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("User Registered"))

}
