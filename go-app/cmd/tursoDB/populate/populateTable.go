package main

import (
	"arc42-status/internal/database"
	"fmt"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"math/rand"
	"time"
)

const RecordCount = 100

var now time.Time

func randomDateTime(monthAgo int) string {

	// get the timestamp for some time ago from now
	someTimeAgo := now.AddDate(0, -1*monthAgo, -10)

	// calculate the difference in seconds
	diff := now.Unix() - someTimeAgo.Unix()

	// generate random difference
	randDiff := rand.Int63n(diff)

	// generate random timestamp within the interval [someTimeAgo, now]
	randTime := someTimeAgo.Add(time.Duration(randDiff) * time.Second)

	// Output randomTime to console in SQL datetime format
	return (randTime.Format(database.DateTimeLayout))
}

func populateTimeOfStatusRequestTable(tblName string, index int) {

	randDateTime := randomDateTime(index)

	// language-SQL
	insertStatement := fmt.Sprintf(
		`INSERT INTO %s (TimeCalled,  RequestIP,	Route ) 
				 VALUES ("%s", "localhost", "ping"); `, tblName, randDateTime)

	_, err := database.GetDB().Exec(insertStatement)
	if err != nil {
		fmt.Printf("Error trying to insert %s:%s\n ", tblName, err)
	}
}

func main() {
	fmt.Println("creating content in tables on Turso")

	fmt.Printf("Inserting %d records into TABLE %s;\n", RecordCount, database.TableTimeOfStatusRequest)

	// get the timestamp for current time (now)
	now = time.Now()

	for i := 0; i <= RecordCount; i++ {
		populateTimeOfStatusRequestTable(database.TableTimeOfStatusRequest, i)
	}

}
