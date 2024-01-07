package main

import (
	"arc42-status/internal/database"
	"bufio"
	"fmt"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"os"
	"strings"
)

func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (Yes!!/N): ", prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return false
		}

		response = strings.TrimSpace(response)
		if response == "Yes!!" {
			return true
		} else if response == "n" || response == "N" || response == "" {
			return false
		}
	}
}

func dropOneTable(tblName string) {
	// language-SQL
	dropStatement := "DROP TABLE IF EXISTS " + tblName + ";"
	fmt.Printf("DROP TABLE %s;\n", tblName)

	_, err := database.GetDB().Exec(dropStatement)
	if err != nil {
		fmt.Printf("Error trying to drop %s:%s\n ", tblName, err)
	}

}

func dropTables() {
	dropOneTable(database.TableTimeOfInvocation)
	dropOneTable(database.TableTimeOfPlausibleCall)
	dropOneTable(database.TableTimeOfGitHubCall)
}

func main() {
	if confirmAction("Are you sure you want to DROP all tables? ") {
		if confirmAction("Really, really sure (think twice, data might get lost)?") {
			dropTables()
		}
	} else {
		fmt.Println("no tables dropped.")
		// Cancel the action or exit
	}
}
