package rules

import (
	"regexp"
	"strings"

	"github.com/johnstarich/sage/ledger"
)

var (
	// Default is the set of rules applied to incoming transaction, always
	Default = Rules{
		category{Negative: true, Category: "expenses:uncategorized"},
		category{
			PayeeContains: containsPattern(
				"'s",
				".*vend.*",
				"airport",
				"american",
				"bagels?",
				"bakery",
				"bar",
				"barbecue",
				"bbq",
				"bean",
				"beverage",
				"bistro",
				"bowl",
				"burger.*",
				"cafe",
				"caffe",
				"candy",
				"catering",
				"chicken",
				"coffee",
				"coffeehouse",
				"cola",
				"counter",
				"cream",
				"crepe",
				"cuisine",
				"custard",
				"deli",
				"diner",
				"donuts?",
				"drive",
				"food",
				"french",
				"gelato",
				"gourmet",
				"grill",
				"house",
				"indian",
				"italian",
				"japanese",
				"juice",
				"kitchen",
				"lunch.*",
				"market",
				"noodle",
				"pasta",
				"pie",
				"pies",
				"pizza",
				"pizzeria",
				"pot",
				"ramen",
				"restaurant",
				"rice",
				"sandwich",
				"shake",
				"shop",
				"snack",
				"soda",
				"soup",
				"spot",
				"stand",
				"sushi",
				"taco",
				"thai",
				"tortilla",
			),
			Negative: true,
			Category: "expenses:shopping:food:restaurants",
		},
		category{Positive: true, Category: "revenues:uncategorized"},
		category{
			PayeeContains: containsPattern(
				"autodiv",
				"dividend",
				"int",
				"interest",
			),
			Positive: true,
			Category: "revenues:interest",
		},
		category{
			PayeeContains: containsPattern(
				"check",
				"deposit",
			),
			Positive: true,
			Category: "revenues:deposits",
		},
		category{
			PayeeContains: containsPattern(
				"transfer",
				"wire",
			),
			Category: "expenses:transfers",
		},
		category{
			PayeeContains: containsPattern(
				"irs",
				"us treasury",
			),
			Positive: true,
			Category: "revenues:tax returns",
		},
		category{
			PayeeContains: containsPattern(
				"dental",
				"optometrist",
				"medical",
			),
			Negative: true,
			Category: "expenses:health",
		},
		category{
			PayeeContains: containsPattern(
				"city",
				"grande",
				"spectrum",
				"comcast",
			),
			Negative: true,
			Category: "expenses:home:utilities",
		},
		category{
			PayeeContains: containsPattern(
				".*\\.com?",
				"amazon",
				"amzn",
				"barnes.*noble",
				"books?",
				"cvs",
				"e-commerce",
				"gift",
				"hallmark",
				"ikea",
				"photo.*",
				"sq \\*",
				"staples",
				"walgreens",
			),
			Negative: true,
			Category: "expenses:shopping",
		},
		category{
			PayeeContains: containsPattern(
				"apple",
				"best buy",
				"computers?",
				"dell",
				"drop",
				"electronics",
				"fry's",
				"gamestop",
				"massdrop",
				"newegg.com",
				"software",
				"steamgames.com",
				"steampowered.com",
				"texas instruments",
			),
			Negative: true,
			Category: "expenses:shopping:electronics",
		},
		category{
			PayeeContains: containsPattern(
				"apple music",
				"audible",
				"codeschool.com",
				"godaddy.com",
				"itunes.com/bill",
				"membership",
				"name-cheap.com",
				"pandora",
				"spotify",
				"subscriptions?",
			),
			Negative: true,
			Category: "expenses:shopping:subscriptions",
		},
		category{
			PayeeContains: containsPattern(
				"7-eleven",
				"chevron",
				"exxon.*",
				"shell",
			),
			Negative: true,
			Category: "expenses:car:gas",
		},
		category{
			PayeeContains: containsPattern(
				"vehreg",
				"dps",
			),
			Negative: true,
			Category: "expenses:car:registration",
		},
		category{
			PayeeContains: containsPattern(
				"autopay",
				"directpay",
				"e-?payment",
				"internet payment",
			),
			Category: "expenses:transfers:credit card payments",
		},
		category{
			PayeeContains: containsPattern(
				"alamo",
				"amc",
				"cinema",
				"cinemark",
				"conv center",
				"fandango.com",
				"fgt\\*",
				"stubhub",
				"ticketfly",
				"tickets?",
			),
			Negative: true,
			Category: "expenses:concerts and shows",
		},
		category{
			PayeeContains: containsPattern(
				"heb",
				"h-e-b",
				"wal-mart",
				"supercenter",
				"wholefds",
				"mart",
				"market",
				"safeway",
				"grocer",
				"liquor",
			),
			Negative: true,
			Category: "expenses:shopping:food:groceries",
		},
	}
)

func containsPattern(strs ...string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b(` + strings.Join(strs, "|") + `)\b`)
}

type category struct {
	// these fields are triggers for a rule
	PayeeContains *regexp.Regexp
	Positive      bool
	Negative      bool
	Zero          bool

	// these fields are applied to the transaction
	Category string
}

// assumes only 2 postings
func (c category) Match(txn ledger.Transaction) bool {
	if c.PayeeContains != nil && !c.PayeeContains.MatchString(txn.Payee) {
		return false
	}
	amt := txn.Postings[0].Amount
	if amt.IsZero() && !c.Zero {
		return false
	}
	if amt.IsPositive() && !c.Positive {
		return false
	}
	if amt.IsNegative() && !c.Negative {
		return false
	}
	// only true if all conditions are met
	return true
}

func (c category) Apply(txn *ledger.Transaction) {
	txn.Postings[len(txn.Postings)-1].Account = c.Category
}
