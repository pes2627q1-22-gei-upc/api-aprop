package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    "github.com/go-sql-driver/mysql"
)

var db *sql.DB

type Test struct{
    ID int
    titol string
    descripcio string
}

func main() {
    // Capture connection properties.
    cfg := mysql.NewConfig()
    cfg.User = os.Getenv("DBUSER")
    cfg.Passwd = os.Getenv("DBPASS")
    cfg.Net = "tcp"
    cfg.Addr = "127.0.0.1:3306"
    cfg.DBName = "test"

    // Get a database handle.
    var err error
    db, err = sql.Open("mysql", cfg.FormatDSN())
    if err != nil {
        log.Fatal(err)
    }

    pingErr := db.Ping()
    if pingErr != nil {
        log.Fatal(pingErr)
    }
    fmt.Println("Connected!")

    tests, err := getAllRows()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found: %v\n", tests)
}

func getAllRows()([]Test, error){
    var tests []Test

    rows, err := db.Query("SELECT * FROM test")
    if err != nil {
        return nil, fmt.Errorf("getAllRows %v", err)
    }
    defer rows.Close()
    // Loop through rows, using Scan to assign column data to struct fields.
    for rows.Next() {
        var t Test
        if err := rows.Scan(&t.ID, &t.titol, &t.descripcio); err != nil {
            return nil, fmt.Errorf("getAllRows %v", err)
        }
        tests = append(tests, t)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("getAllRows %v", err)
    }
    return tests, nil
}