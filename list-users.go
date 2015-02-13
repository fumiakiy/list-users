package main

import (
    "database/sql"
    "encoding/json"
    "encoding/csv"
    "log"
    "net/http"
    "net/url"
    "html/template"
    "os"
    _ "github.com/go-sql-driver/mysql"
    "time"
)

type Configuration struct {
    Dsn   string
    Sql1   string
    Sql2   string
}

type User struct {
    Email string
    Nickname string
    Language string
    EventName string
}

var templates = template.Must(template.ParseFiles("index.html"))
var config Configuration

func readConfig() error {
    config = Configuration{}
    file, err := os.Open("conf.json")
    if err != nil {
        return err
    }

    decoder := json.NewDecoder(file)
    err = decoder.Decode(&config)
    if err != nil {
        return err;
    }

    return nil;
}

func getUsers(sqlString string, countryCode string, since time.Time) ([]User, error) {
    var users []User

    db, err := sql.Open("mysql", config.Dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    stmt, err := db.Prepare(sqlString)
    if err != nil {
        log.Fatal(err)
    }

    var args []interface{}
    args = make([]interface{}, 2)
    args[0] = countryCode
    args[1] = since.Format("2006-01-02") + " 00:00:00"

    rows, err := stmt.Query(args...)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    user := User{}
    for rows.Next() {
        err := rows.Scan(
            &user.Email,
            &user.Nickname,
            &user.Language,
            &user.EventName,
        )
        if err != nil {
            log.Fatal(err)
        }
        users = append(users, user)
    }
    err = rows.Err()
    if err != nil {
        log.Fatal(err)
    }

    return users, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    err := templates.ExecuteTemplate(w, "index.html", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func attendeesHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    sinceStr := query["since"][0]
    if len(sinceStr) < 10 {
        http.Error(w, "Enter since date in YYYY-MM-DD format", http.StatusBadRequest)
        return;
    }
    countryStr := query["country"][0]
    if len(countryStr) < 2 {
        http.Error(w, "Select a country", http.StatusBadRequest)
        return;
    }
    since, err := time.Parse("2006-01-02", sinceStr)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return;
    }

    users, err := getUsers(config.Sql1, countryStr, since)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-disposition", "attachment; filename=" + url.QueryEscape("Attendees-" + countryStr + "-" + sinceStr + ".csv"))

    csvWriter := csv.NewWriter(w)

    for _, u := range users {
        _ = csvWriter.Write([]string { u.Email, u.Nickname, u.Language, u.EventName })
    }
    csvWriter.Flush()
}

func organizersHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    sinceStr := query["since"][0]
    countryStr := query["country"][0]
    since, err := time.Parse("2006-01-02", sinceStr)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
    }

    users, err := getUsers(config.Sql2, countryStr, since)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-disposition", "attachment; filename=" + url.QueryEscape("Organizers-" + countryStr + "-" + sinceStr + ".csv"))

    csvWriter := csv.NewWriter(w)

    for _, u := range users {
        _ = csvWriter.Write([]string { u.Email, u.Nickname, u.Language, u.EventName })
    }
    csvWriter.Flush()
}

func main() {
    err := readConfig()
    if err != nil {
        log.Fatal(err)
    }

    //var page int
    //flag.IntVar(&page, "page", 1, "number of page to search for (starts 1)")
    //var eid int
    //flag.IntVar(&eid, "id", 0, "event id")

    //flag.Parse()

    //if len(flag.Args()) <= 0 {
    //    log.Fatal("keyword required")
    //}
    //keyword := flag.Args()[0]

    http.HandleFunc("/", indexHandler)
    http.HandleFunc("/attendee.csv", attendeesHandler)
    http.HandleFunc("/organizer.csv", organizersHandler)
    http.ListenAndServe(":8080", nil)
}

