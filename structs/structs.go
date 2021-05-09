package structs

type BrePkg struct {
	
	PkgCode   string    `json:"pkgCode"`
	Cat       string    `json:"cat"`
	Site      string    `json:"site"`
	User      string    `json:"user"`
	ValidFrom string    `json:"validFrom"`
	ValidTo   string    `json:"validTo"`
	RuleSet   []ruleSet `json:"ruleSet"`
	Filters   []string  `json:"filters"`
}

type ruleSet struct {
	RuleName string   `json1:"ruleName"`
	Rule     string   `json:"rule"`
	Actions  []string `json:"actions"`
}

// Database table user row struct
type User struct {
	UserPk    int
	UserId    string `json:"userId"`
	PswdHash  string `json:"pswdHash"`
	Name      string
	Sbu       string
	Email     string
	MobileNbr string
	IpAddress string
}
