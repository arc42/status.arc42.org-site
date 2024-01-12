package main

import (
	"arc42-status/internal/database"
	"bufio"
	"fmt"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"os"
	"strings"
)

func chooseTable() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Please select which table you like to see:")
	fmt.Printf("1. %s\n", database.TableTimeOfInvocation)
	fmt.Printf("2. %s\n", database.TableTimeOfPlausibleCall)
	fmt.Printf("3. %s\n", database.TableTimeOfGitHubCall)
	fmt.Print("Enter choice (1-3): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("An error occurred while reading input. Please try again", err)
		return ""
	}

	input = strings.TrimSpace(input)

	switch input {
	case "1":
		return database.TableTimeOfInvocation
	case "2":
		return database.TableTimeOfPlausibleCall
	case "3":
		return database.TableTimeOfGitHubCall
	default:
		return ""
	}
}

func selectOneTable(tblName string) {
	// language-SQL
	selectStatement := "SELECT TimeCalled, Route FROM " + tblName + ";"
	fmt.Printf("Selecting from TABLE %s;\n", tblName)

	rows, err := database.GetDB().Query(selectStatement)
	if err != nil {
		fmt.Printf("Error trying to select %s:%s\n ", tblName, err)
	} else {
		// Iterate over the rows
		i := 0
		for rows.Next() {
			i++
			var timeCalled, route string
			err := rows.Scan(&timeCalled, &route)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("%d: Time called: %s, column2: %s\n", i, timeCalled, route)
		}
	}

}

func main() {
	fmt.Println("dumping content of tables from Turso")
	tableToDump := chooseTable()

	if tableToDump == "" {
		fmt.Println("nothing selected, aborting.")
	} else {
		selectOneTable(tableToDump)
	}
}
