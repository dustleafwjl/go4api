/*
 * go4api - a api testing tool written in Go
 * Created by: Ping Zhu 2018
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package gsql

import (
    "os"
    "strconv"
    "fmt"
    // "time"
    // "log"
    "strings"
    "database/sql"

    "go4api/cmd"

    _ "github.com/go-sql-driver/mysql"
)

var db = &sql.DB{}

func InitConnection (ip string, port string, user string, pw string, defaultDB string) {
    conInfo := user + ":" + pw + "@tcp(" + ip + ":" + port + ")/" + defaultDB
    db, _ = sql.Open("mysql", conInfo)

    err := db.Ping()
    if err != nil {
        panic(err)
    }
} 

func Run (stmt string) string {
    // update, delete, select, insert
    s := strings.TrimSpace(stmt)
    s = strings.ToUpper(s)
    s = string([]rune(stmt)[:6])

    var err error

    switch s {
        case "UPDATE":
            Update()
        case "DELETE":
            err = Delete(s)
        case "SELECT":
            QueryWithoutParams()
        case "INSERT":
            Insert() 
    }

    if err != nil {
        return "SqlSuccess"
    } else {
        return "SqlFailed"
    }

}

func Update () {
    tx, _ := db.Begin()
    
    tx.Exec("Update user set age = ? where uid = ?", 1, 1)

    tx.Commit()
}

func Delete (stmt string) error {
    tx, _ := db.Begin()

    tx.Exec(stmt)

    err := tx.Commit()

    return err
}

func QueryWithoutParams () {
    tx, _ := db.Begin()

    tx.Exec("SELECT * FROM STORE;")

    tx.Commit()
}

func QueryWithParams () {
    stm, _ := db.Prepare("SELECT * FROM STORE;")
    defer stm.Close()
    rows, _ := stm.Query()
    defer rows.Close()

    columns, _ := rows.Columns()
    scanArgs := make([]interface{}, len(columns))
    values := make([]interface{}, len(columns))
    for i := range values {
        scanArgs[i] = &values[i]
    }
     
    for rows.Next() {
        rows.Scan(scanArgs...)
        record := make(map[string]string)
        for i, col := range values {
            if col != nil {
                record[columns[i]] = string(col.([]byte))
            }
        }
        fmt.Println(record)
    }
}

func Insert () {
    tx,_ := db.Begin()
    
    tx.Exec("INSERT INTO user(uid, username, age) values(?, ?, ?)", 1, "user" + strconv.Itoa(1), 1)

    tx.Commit()
}

func GetDBConnInfo () (string, string, string, string, string) {
    var ip, port, user, pw, defaultDB string

    testEnv := ""
    if cmd.Opt.TestEnv != "" {
        testEnv = cmd.Opt.TestEnv
    } else {
        testEnv = "QA"
    }

    switch strings.ToLower(testEnv) {
        case "qa":
            ip = os.Getenv("go4_qa_db_ip")
            port = os.Getenv("go4_qa_db_port")
            user = os.Getenv("go4_qa_db_username")
            pw = os.Getenv("go4_qa_db_password")
            defaultDB = os.Getenv("go4_qa_db_defaultDB")
        case "dev":
            ip = os.Getenv("go4_dev_db_ip")
            port = os.Getenv("go4_dev_db_port")
            user = os.Getenv("go4_dev_db_username")
            pw = os.Getenv("go4_dev_db_password")
            defaultDB = os.Getenv("go4_dev_db_defaultDB")
    }

    return ip, port, user, pw, defaultDB
}